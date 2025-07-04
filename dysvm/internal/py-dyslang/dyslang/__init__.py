import ast
import inspect
import itertools
import operator as op
import re
import sys
import types
import typing
from collections import Counter, defaultdict
from collections import namedtuple as nt
from collections.abc import MutableMapping
from copy import copy
from functools import partial

import forge

# Monkey patch the CallArguments to not shadow 'self' in kwargs
# because CallArguments.from_bound_arguments calls CallArguments.__init__
# with the bound arguments, which might include 'self' as a kwarg
orig_init = forge._utils.CallArguments.__init__


def new_init(_forge_self, *args: typing.Any, **kwargs: typing.Any) -> None:
    super(forge._utils.CallArguments, _forge_self).__init__(
        args=args, kwargs=types.MappingProxyType(kwargs)
    )


# bind the new init to the class
forge._utils.CallArguments.__init__ = new_init

########################################
# Module wide 'globals'
MAX_STRING_LENGTH = 100000
MAX_POWER = 100 * 100  # highest exponent
MAX_SCOPE_SIZE = MAX_STRING_LENGTH * 2
MAX_NODE_CALLS = 10000
MAX_CALL_DEPTH = 32
DISALLOW_PREFIXES = ["_"]
DISALLOW_METHODS = [(str, "format"), (type, "mro"), (str, "format_map")]


# Disallow functions:
# This, strictly speaking, is not necessary.  These /should/ never be accessable anyway,
# if DISALLOW_PREFIXES and DISALLOW_METHODS are all right.  This is here to try and help
# people not be stupid.  Allowing these functions opens up all sorts of holes - if any of
# their functionality is required, then please wrap them up in a safe container.  And think
# very hard about it first.  And don't say I didn't warn you.


########################################
# Defaults for the evaluator:

BUILTIN_EXCEPTIONS = {
    "ArithmeticError": ArithmeticError,
    "AssertionError": AssertionError,
    "AttributeError": AttributeError,
    "Exception": Exception,
    "FloatingPointError": FloatingPointError,
    "ImportError": ImportError,
    "IndexError": IndexError,
    "KeyError": KeyError,
    "LookupError": LookupError,
    "ModuleNotFoundError": ModuleNotFoundError,
    "NameError": NameError,
    "NotImplementedError": NotImplementedError,
    "OverflowError": OverflowError,
    "PermissionError": PermissionError,
    "RecursionError": RecursionError,
    "SyntaxError": SyntaxError,
    "TypeError": TypeError,
    "UnboundLocalError": UnboundLocalError,
    "UnicodeDecodeError": UnicodeDecodeError,
    "UnicodeEncodeError": UnicodeEncodeError,
    "UnicodeError": UnicodeError,
    "UnicodeTranslateError": UnicodeTranslateError,
    "ValueError": ValueError,
    "ZeroDivisionError": ZeroDivisionError,
    "MemoryError": MemoryError,
}


DISALLOW_FUNCTIONS = {
    type,
    eval,
    getattr,
    setattr,
    help,
    repr,
    compile,
    open,
    exec,
    format,
    vars,
}


def assert_func_allowed(func):
    try:
        if not callable(func):
            # If not callable, skip whitelist check
            return
            
        modname = getattr(func, "__module__", None)
        qualname = getattr(
            func, "__qualname__", getattr(func, "__name__", getattr(func, "_name", None))
        )

        if modname and qualname:
            fullname = modname + "." + qualname
        else:
            fullname = qualname or "unknown"

        if func in DISALLOW_FUNCTIONS:
            raise DangerousValue(f"This function is forbidden: {fullname}")

        if fullname not in WHITELIST_FUNCTIONS:
            raise NotImplementedError(
                "Creativity needs constraint. When the constraints get tighter, you will find more opportunities to be creative. This function is not allowed: '{}'".format(
                    fullname
                )
            )
    except (AttributeError, TypeError):
        # If there was an error getting attributes, be safe and disallow
        raise DangerousValue("Unable to validate function permissions")


########################################
# Exceptions:


class Return(Exception):
    """Not actually an exception, just a way to break out of the function"""

    def __init__(self, value):
        self.value = value


class Break(Exception):
    """Not actually an exception, just a way to break out of the loop"""


class Continue(Exception):
    """Not actually an exception, just a way to continue the loop"""


class InvalidExpression(Exception):
    """Generic Exception"""

    pass


class DangerousValue(Exception):
    """When you try to pass in something dangerous to dys, it won't catch everything though"""

    def __init__(self, *args):
        super().__init__(*args)


class DysRuntimeError(Exception):
    """Something caused the Dys code to crash"""

    lineno = None
    col_offset = None
    end_lineno = None
    end_col_offset = None
    col = None

    def __init__(self, msg, node=None):
        self.__node = node
        if node:  # pragma: no branch
            self.lineno = getattr(self.__node, "lineno", None)
            self.col_offset = getattr(self.__node, "col_offset", None)
            self.end_lineno = getattr(self.__node, "end_lineno", None)
            self.end_col_offset = getattr(self.__node, "end_col_offset", None)
            self.col = getattr(self.__node, "col_offset", None)

        super().__init__(msg)


UNCATCHABLE_EXCEPTIONS = (DangerousValue, MemoryError)
########################################
# Default simple functions to include:


def safe_mod(a, b):
    """only allow modulo on numbers, not string formating"""
    if isinstance(a, str):
        raise NotImplementedError("String formating is not supported")
    return a % b


def safe_power(a, b):  # pylint: disable=invalid-name
    """a limited exponent/to-the-power-of function, for safety reasons"""

    if abs(a) > MAX_POWER or abs(b) > MAX_POWER:
        raise MemoryError("Sorry! I don't want to evaluate {0} ** {1}".format(a, b))
    return a**b


def safe_mult(a, b):  # pylint: disable=invalid-name
    """limit the number of times an iterable can be repeated..."""
    if hasattr(a, "__len__") and b * len(str(a)) >= MAX_SCOPE_SIZE:
        raise MemoryError("Sorry, I will not evalute something that long.")
    if hasattr(b, "__len__") and a * len(str(b)) >= MAX_SCOPE_SIZE:
        raise MemoryError("Sorry, I will not evalute something that long.")

    return a * b


def safe_add(a, b):  # pylint: disable=invalid-name
    """iterable length limit again"""
    if hasattr(a, "__len__") and hasattr(b, "__len__"):
        if len(a) + len(b) > MAX_STRING_LENGTH:
            raise MemoryError(
                "Sorry, adding those two together would make something too long."
            )
    return a + b


DEFAULT_SCOPE = {
    "Exception": Exception,
    "False": False,
    "None": None,
    "True": True,
    "abs": abs,
    "all": all,
    "any": any,
    "bin": bin,
    "bool": bool,
    "bytearray": bytearray,
    "bytes": bytes,
    "callable": callable,
    "chr": chr,
    "classmethod": classmethod,
    "complex": complex,
    "dict": dict,
    "divmod": divmod,
    "enumerate": enumerate,
    "filter": filter,
    "frozenset": frozenset,
    "hex": hex,
    "int": int,
    "isinstance": isinstance,
    "issubclass": issubclass,
    "iter": iter,
    "len": len,
    "list": list,
    "map": map,
    "max": max,
    "min": min,
    "oct": oct,
    "ord": ord,
    "pow": safe_power,
    "range": range,
    "reversed": reversed,
    "round": round,
    "set": set,
    "slice": slice,
    "sorted": sorted,
    "str": str,
    "sum": sum,
    "tuple": tuple,
    "zip": zip,
    **BUILTIN_EXCEPTIONS,
}

_whitelist_functions_dict = {
    "str": ["join"],
    "builtins": [*list(DEFAULT_SCOPE.keys())],
    "dict": ["get", "items", "keys", "values"],
    "list": ["sort", "append", "pop", "count", "index", "reverse"],
}


WHITELIST_FUNCTIONS = set()

for k, v in _whitelist_functions_dict.items():
    for kk in v:
        WHITELIST_FUNCTIONS.add(k + "." + kk)


def make_modules(mod_dict, k=""):
    if not isinstance(mod_dict, dict):
        return mod_dict
    ret = types.ModuleType(k)
    for kk, vv in mod_dict.items():
        ret.__dict__[kk] = make_modules(vv, kk)
    return ret


class Scope(MutableMapping):
    __slots__ = ("dicts",)

    def __init__(self, mapping=(), **kwargs):
        self.dicts = [kwargs]

    def __len__(self):
        return [len(d) for d in self.dicts]

    def __repr__(self):
        return repr(self.dicts[1:])

    def __copy__(self):
        duplicate = copy(super())
        duplicate.dicts = self.dicts[:]
        return duplicate

    def push(self, d):
        self.dicts.append(d)

    def __iter__(self):
        return iter(self.flatten())

    def __setitem__(self, key, value):
        self.dicts[-1][key] = value

    def __getitem__(self, key):
        for d in reversed(self.dicts):
            if key in d:
                return d[key]
        raise KeyError(key)

    def __delitem__(self, key):
        del self.dicts[-1][key]

    def __contains__(self, key):
        return any(key in d for d in self.dicts)

    def flatten(self):
        flat = {}
        for d in self.dicts:
            flat.update(d)
        return flat

    def update(self, other_dict):
        return self.dicts[-1].update(other_dict)

    def locals(self):
        return self.dicts[-1]

    def globals(self):
        return self.dicts[1]  # layer 0 is builtins


class DysEval(object):
    nodes_called = 0
    # temp place to track return values
    _last_eval_result = None

    def __init__(
        self,
        scope=None,
        modules=None,
        call_stack=None,
        local_track_func=None,
        max_scope_size=None,
        max_node_calls=None,
    ):

        if call_stack is None:
            call_stack = []
        self.call_stack = call_stack

        self.local_track_func = local_track_func
        self.max_scope_size = min(max_scope_size or MAX_SCOPE_SIZE, MAX_SCOPE_SIZE)
        self.max_node_calls = min(max_node_calls or MAX_NODE_CALLS, MAX_NODE_CALLS)

        self.operators = {
            ast.Add: safe_add,
            ast.Sub: op.sub,
            ast.Mult: safe_mult,
            ast.Div: op.truediv,
            ast.FloorDiv: op.floordiv,
            ast.Pow: safe_power,
            ast.Mod: safe_mod,
            ast.Eq: op.eq,
            ast.NotEq: op.ne,
            ast.Gt: op.gt,
            ast.Lt: op.lt,
            ast.GtE: op.ge,
            ast.LtE: op.le,
            ast.Not: op.not_,
            ast.USub: op.neg,
            ast.UAdd: op.pos,
            ast.Invert: op.invert,
            ast.In: lambda x, y: op.contains(y, x),
            ast.NotIn: lambda x, y: not op.contains(y, x),
            ast.Is: op.is_,
            ast.IsNot: op.is_not,
            ast.BitOr: op.or_,
            ast.BitXor: op.xor,
            ast.BitAnd: op.and_,
            ast.LShift: op.lshift,
            ast.RShift: op.rshift,
        }

        if scope is None:
            scope = {}

        self.scope = Scope()
        self.scope.update(DEFAULT_SCOPE)
        self.scope.push(scope)

        self.modules = {}
        if modules is not None:
            self.modules = modules

        self.nodes = {
            ast.Constant: self._eval_constant,
            ast.Num: self._eval_num,
            ast.Bytes: self._eval_bytes,
            ast.Str: self._eval_str,
            ast.Name: self._eval_name,
            ast.UnaryOp: self._eval_unaryop,
            ast.BinOp: self._eval_binop,
            ast.BoolOp: self._eval_boolop,
            ast.Compare: self._eval_compare,
            ast.IfExp: self._eval_ifexp,
            ast.If: self._eval_if,
            ast.Try: self._eval_try,
            ast.ExceptHandler: self._eval_excepthandler,
            ast.Call: self._eval_call,
            ast.keyword: self._eval_keyword,
            ast.Subscript: self._eval_subscript,
            ast.Attribute: self._eval_attribute,
            ast.Index: self._eval_index,
            ast.Slice: self._eval_slice,
            ast.Module: self._eval_module,
            ast.Expr: self._eval_expr,
            ast.AugAssign: self._eval_augassign,
            ast.Assign: self._eval_assign,
            ast.AnnAssign: self._eval_annassign,
            ast.NamedExpr: self._eval_namedexpr,
            ast.Lambda: self._eval_lambda,
            ast.FunctionDef: self._eval_functiondef,
            ast.arguments: self._eval_arguments,
            ast.Return: self._eval_return,
            ast.ClassDef: self._eval_classdef,
            ast.JoinedStr: self._eval_joinedstr,  # f-string
            ast.NameConstant: self._eval_nameconstant,
            ast.FormattedValue: self._eval_formattedvalue,
            ast.Dict: self._eval_dict,
            ast.Tuple: self._eval_tuple,
            ast.List: self._eval_list,
            ast.Set: self._eval_set,
            ast.ListComp: self._eval_comprehension,
            ast.SetComp: self._eval_comprehension,
            ast.DictComp: self._eval_comprehension,
            # ast.GeneratorExp: self._eval_comprehension,
            ast.ImportFrom: self._eval_importfrom,
            ast.Import: self._eval_import,
            ast.For: self._eval_for,
            ast.While: self._eval_while,
            ast.Break: self._eval_break,
            ast.Continue: self._eval_continue,
            ast.Pass: self._eval_pass,
            ast.Assert: self._eval_assert,
            ast.Delete: self._eval_delete,
            ast.Raise: self._eval_raise,
            # not really none, these are handled differently
            ast.And: None,
            ast.Or: None,
            ast.Store: None,  # Not used, needed for validation
        }

        self.assignments = {
            ast.Name: self._assign_name,
            ast.Tuple: self._assign_tuple_or_list,
            ast.Subscript: self._assign_subscript,
            ast.List: self._assign_tuple_or_list,
            ast.Starred: self._assign_starred,
        }

        self.deletions = {
            ast.Name: self._delete_name,
            ast.Subscript: self._delete_subscript,
        }

        # Check for forbidden functions:
        for name, func in self.scope.flatten().items():
            if callable(func):
                try:
                    hash(func)
                except TypeError:
                    raise DangerousValue(
                        "This function '{}' in scope might be a bad idea.".format(name)
                    )
                if func in DISALLOW_FUNCTIONS:
                    raise DangerousValue(
                        "This function '{}' in scope is {} and is in DISALLOW_FUNCTIONS".format(
                            name, func
                        )
                    )

    def validate(self, expr):
        """Validate that all ast.Nodes are supported by this sandbox"""
        tree = ast.parse(expr)
        ignored_nodes = set(
            [
                ast.Load,
                ast.Del,
                ast.Starred,
                ast.arg,
                ast.comprehension,
                ast.alias,
                ast.GeneratorExp,
                ast.ListComp,
                ast.DictComp,
                ast.SetComp,
            ]
        )
        valid_nodes = (
            ignored_nodes
            | set(self.nodes)
            | set(self.operators)
            | set(self.deletions)
            | set(self.assignments)
        )
        for node in ast.walk(tree):
            if node.__class__ not in valid_nodes:
                exc = NotImplementedError(
                    f"Sorry, {node.__class__.__name__} is not available in this evaluator"
                )
                exc._dys_node = node
                raise exc
        compile(expr, "<string>", "exec", dont_inherit=True)

    def eval(self, expr):
        """evaluate an expresssion, using the operators, functions and
        scope previously set up."""

        # set a copy of the expression aside, so we can give nice errors...
        self.expr = expr
        try:
            self.validate(expr)
        except (Exception,) as e:
            exc = e
            node = None
            if hasattr(exc, "_dys_node"):  # pragma: no branch
                node = exc._dys_node
            raise DysRuntimeError(repr(exc), node=node) from exc

        # and evaluate:
        return self._eval(ast.parse(expr))

    def _eval(self, node):
        """The internal evaluator used on each node in the parsed tree."""
        self.nodes_called += 1
        if self.nodes_called > self.max_node_calls:
            raise Exception(
                f"This script exceeded the maximum number of allowed node calls: {self.max_node_calls}"
            )
        try:
            try:
                lineno = getattr(node, "lineno", None)  # noqa: F841
                col_offset = getattr(node, "col_offset", None)  # noqa: F841
                end_lineno = getattr(node, "end_lineno", None)  # noqa: F841
                end_col_offset = getattr(node, "end_col_offset", None)  # noqa: F841

                try:
                    handler = self.nodes[type(node)]
                except KeyError:
                    raise NotImplementedError(
                        "Sorry, {0} is not available in this "
                        "evaluator".format(type(node).__name__)
                    )
                node.call_stack = self.call_stack
                self._last_eval_result = handler(node)

                if callable(self._last_eval_result):
                    assert_func_allowed(self._last_eval_result)

                return self._last_eval_result
            finally:
                self.track(node)
                if self.local_track_func:
                    self.local_track_func(
                        node.lineno,
                        node.col_offset,
                        node.end_lineno,
                        node.end_col_offset,
                        node.__class__.__name__,
                    )
        except (Return, Break, Continue, DysRuntimeError):
            raise
        except Exception as e:
            exc = e
            if not hasattr(exc, "_dys_node"):  # pragma: no branch
                exc._dys_node = node
            #raise e
            raise DysRuntimeError(repr(exc), node=node) from exc

    def _eval_assert(self, node):
        if not self._eval(node.test):
            if node.msg:
                raise AssertionError(self._eval(node.msg))
            raise AssertionError()

    def _eval_while(self, node):
        while self._eval(node.test):
            try:
                for b in node.body:
                    self._eval(b)
            except Break:
                break
            except Continue:
                continue
        else:
            for b in node.orelse:
                self._eval(b)

    def _eval_for(self, node):
        def recurse_targets(target, value):
            """
                Recursively (enter, (into, (nested, name), unpacking)) = \
                             and, (assign, (values, to), each
            """
            self.track(target)
            if isinstance(target, ast.Name):
                self.scope[target.id] = value
            else:
                for t, v in zip(target.elts, value):
                    recurse_targets(t, v)

        for v in self._eval(node.iter):
            recurse_targets(node.target, v)
            try:
                for b in node.body:
                    self._eval(b)
            except Break:
                break
            except Continue:
                continue
        else:
            for b in node.orelse:
                self._eval(b)

    def _eval_import(self, node):
        if not self.modules:
            raise ModuleNotFoundError(alias.name)
        for alias in node.names:
            self.track(alias)
            asname = alias.asname or alias.name.split(".")[0]
            m = self.modules
            try:
                m = m.__dict__[alias.name.split(".")[0]]
            except KeyError:
                raise ModuleNotFoundError(alias.name)
            self.scope[asname] = m

    def _eval_importfrom(self, node):
        if not self.modules:
            raise ModuleNotFoundError(alias.name)
            
        for alias in node.names:
            self.track(alias)
            asname = alias.asname or alias.name
            try:
                module = self.modules
                for name in node.module.split("."):
                    module = module.__dict__[name]
            except KeyError:
                raise ModuleNotFoundError(node.module)
            if alias.name == "*":
                self.scope.update(module.__dict__)
            else:
                try:
                    submodule = module.__dict__[alias.name]
                    self.scope[asname] = submodule
                except KeyError:
                    raise ImportError(alias.name)

    def _eval_expr(self, node):
        return self._eval(node.value)

    def _eval_module(self, node):
        return [self._eval(b) for b in node.body]

    def _eval_arguments(self, node):
        NONEXISTANT_DEFAULT = object()  # a unique object to contrast with None
        posonlyargs_and_defaults = []
        num_args = len(node.args)
        if hasattr(node, "posonlyargs"):
            for arg, default in itertools.zip_longest(
                node.posonlyargs[::-1],
                node.defaults[::-1][num_args:],
                fillvalue=NONEXISTANT_DEFAULT,
            ):
                self.track(arg)
                if default is NONEXISTANT_DEFAULT:
                    posonlyargs_and_defaults.append(forge.pos(arg.arg))
                else:
                    posonlyargs_and_defaults.append(
                        forge.pos(arg.arg, default=self._eval(default))
                    )
            posonlyargs_and_defaults.reverse()

        args_and_defaults = []
        for arg, default in itertools.zip_longest(
            node.args[::-1],
            node.defaults[::-1][:num_args],
            fillvalue=NONEXISTANT_DEFAULT,
        ):
            self.track(arg)
            if default is NONEXISTANT_DEFAULT:
                args_and_defaults.append(forge.arg(arg.arg))
            else:
                args_and_defaults.append(
                    forge.arg(arg.arg, default=self._eval(default))
                )
        args_and_defaults.reverse()
        vpo = (node.vararg and forge.args(node.vararg.arg)) or []

        kwonlyargs_and_defaults = []
        # kwonlyargs is 1:1 to kw_defaults, no need to jump through hoops
        for arg, default in zip(node.kwonlyargs, node.kw_defaults):
            self.track(arg)
            if not default:
                kwonlyargs_and_defaults.append(forge.kwo(arg.arg))
            else:
                kwonlyargs_and_defaults.append(
                    forge.kwo(arg.arg, default=self._eval(default))
                )
        vkw = (node.kwarg and forge.kwargs(node.kwarg.arg)) or {}

        return (
            [
                *posonlyargs_and_defaults,
                *args_and_defaults,
                *vpo,
                *kwonlyargs_and_defaults,
            ],
            vkw,
        )

    def _eval_break(self, node):
        raise Break()

    def _eval_continue(self, node):
        raise Continue()

    def _eval_pass(self, node):
        pass

    def _eval_return(self, node):
        ret = None
        if node.value is not None:
            ret = self._eval(node.value)
        raise Return(ret)

    def _eval_lambda(self, node):

        sig_list, sig_dict = self._eval(node.args)
        _class = self.__class__

        def _func(*args, **kwargs):
            local_scope = {
                inspect.getfullargspec(_func).varargs: args,
                **{
                    kwo: kwargs.pop(kwo)
                    for kwo in inspect.getfullargspec(_func).kwonlyargs
                    + inspect.getfullargspec(_func).args
                },
                inspect.getfullargspec(_func).varkw: kwargs,
            }
            s = _class(
                modules=self.modules,
                scope=copy(self.scope),
                call_stack=self.call_stack,
                local_track_func=self.local_track_func,
                max_scope_size=self.max_scope_size,
                max_node_calls=self.max_node_calls,
            )
            s.scope.push(local_scope)
            s.expr = self.expr
            s.track = self.track
            return s._eval(node.body)

        _func = forge.sign(*sig_list, **sig_dict)(_func)
        del _func.__wrapped__
        _func.__name__ = "<lambda>"
        _func.__qualname__ = "<lambda>"
        _func.__module__ = "script"
        WHITELIST_FUNCTIONS.add(f"{_func.__module__}.{_func.__qualname__}")

        return _func

    def _eval_functiondef(self, node):

        sig_list, sig_dict = self._eval(node.args)
        _annotations = {}
        for a in node.args.args + getattr(node.args, "posonlyargs", [None]) + getattr(node.args, "kwonlyargs", [None]) + [node.args.kwarg]:
            self.track(a)
            if a and a.annotation:
                _annotations[a.arg] = self._eval(a.annotation)
        
        if node.returns:
            _annotations["return"] = self._eval(node.returns)

        _class = self.__class__

        def _func(*args, **kwargs):
            # reconostruct what the orignial function arguments would have been
            local_scope = {
                inspect.getfullargspec(_func).varargs: args,
                **{
                    kwo: kwargs.pop(kwo)
                    for kwo in inspect.getfullargspec(_func).kwonlyargs
                    + inspect.getfullargspec(_func).args
                },
                inspect.getfullargspec(_func).varkw: kwargs,
            }
            s = _class(
                modules=self.modules,
                scope=copy(self.scope),
                call_stack=self.call_stack,
                local_track_func=self.local_track_func,
                max_scope_size=self.max_scope_size,
                max_node_calls=self.max_node_calls,
            )
            s.scope.push(local_scope)
            s.expr = self.expr
            # s.track = self.track
            for b in node.body:
                try:
                    s._eval(b)
                except Return as r:
                    return r.value

        _func.__name__ = node.name
        _func.__qualname__ = node.name
        _func.__module__ = "script"
        _func.__annotations__ = _annotations
        # _func.__qualname__ = node.name
        _func = forge.sign(*sig_list, **sig_dict)(_func)
        del _func.__wrapped__
        WHITELIST_FUNCTIONS.add(f"{_func.__module__}.{_func.__qualname__}")

        # prevent unwrap from detecting this nested function
        _func.__doc__ = ast.get_docstring(node)

        decorated_func = _func
        decorators = [self._eval(d) for d in node.decorator_list]
        for decorator in decorators[::-1]:
            decorated_func = decorator(decorated_func)

        self.scope[node.name] = decorated_func

    def _eval_classdef(self, node):
        # Evaluate base classes
        bases = [self._eval(base) for base in node.bases]

        # Evaluate keywords
        if node.keywords:
            raise NotImplementedError(
                "Sorry, metaclass keyword arguments are not supported"
            )

        # Create new local scope and evaluate the body
        class_scope_dict = {}
        self.scope.push(class_scope_dict)

        docstring = ast.get_docstring(node)
        if docstring is not None:
            class_scope_dict["__doc__"] = docstring

        try:
            for stmt in node.body:
                self._eval(stmt)
        finally:
            self.scope.dicts.pop()

        # Evaluate decorators
        decorators = [self._eval(decorator) for decorator in node.decorator_list]

        # Create the class
        cls = type(node.name, tuple(bases), class_scope_dict)

        # Apply decorators
        for decorator in reversed(decorators):
            cls = decorator(cls)

        cls.__name__ = node.name
        cls.__module__ = "script"
        cls.__qualname__ = node.name
        WHITELIST_FUNCTIONS.add(f"{cls.__module__}.{cls.__qualname__}")
        # Pop local scope

        # Assign the class to its name
        self.scope[node.name] = cls

    def _assign_tuple_or_list(self, node, values):
        try:
            iter(values)
        except TypeError:
            raise TypeError(
                f"cannot unpack non-iterable { type(values).__name__ } object"
            )
        len_elts = len(node.elts)
        len_values = len(values)
        starred_indexes = [
            i for i, n in enumerate(node.elts) if isinstance(n, ast.Starred)
        ]
        if len(starred_indexes) == 1:
            if len_elts - 1 > len_values:
                raise ValueError(
                    f"not enough values to unpack (expected at least { len_elts - 1 }, got { len_values })"
                )
            starred_index = starred_indexes[0]
            before_slice = slice(
                starred_index, len_values - (len_elts - starred_index - 1)
            )
            after_slice = slice(len_values - (len_elts - starred_index - 1), len_values)
            starred_values = (
                *values[:starred_index],
                list(values[before_slice]),
                *values[after_slice],
            )
            for target, value in zip(node.elts, starred_values):
                handler = self.assignments[type(target)]
                handler(target, value)

        else:
            if len_elts > len_values:
                raise ValueError(
                    f"not enough values to unpack (expected { len_elts }, got { len_values })"
                )
            elif len_elts < len_values:
                raise ValueError(f"too many values to unpack (expected { len_elts })")

            for target, value in zip(node.elts, values):
                handler = self.assignments[type(target)]
                handler(target, value)

    def _assign_name(self, node, value):
        self.scope[node.id] = value
        return value

    def _assign_subscript(self, node, value):
        _slice = self._eval(node.slice)
        self._eval(node.value)[_slice] = value
        return value

    def _assign_starred(self, node, value):
        return self._assign([node.value], value)

    def _delete(self, targets):
        if len(targets) > 1:
            raise NotImplementedError(
                "Sorry, cannot delete {} targets.".format(len(targets))
            )
        target = targets[0]
        try:
            handler = self.deletions[type(target)]
            handler(target)
        except KeyError:
            raise NotImplementedError(
                "Sorry, cannot delete {}".format(type(target).__name__)
            )

    def _delete_name(self, node):
        del self.scope[node.id]

    def _delete_subscript(self, node):
        _slice = self._eval(node.slice)
        del self._eval(node.value)[_slice]

    def _eval_delete(self, node):
        return self._delete(node.targets)

    def _eval_raise(self, node):
        exc = self._eval(node.exc)
        exc.node = node
        if node.cause is not None:
            cause = self._eval(node.cause)
            raise exc from cause
        raise exc

    def _assign(self, targets, value):
        for target in targets:
            try:
                handler = self.assignments[type(target)]
                self.track(target)
            except KeyError:  # pragma: no cover
                # This is caught in validate()
                raise NotImplementedError(
                    "Sorry, cannot assign to {0}".format(type(target).__name__)
                )
            handler(target, value)

    def _eval_augassign(self, node):
        if (
            len(self.scope.dicts) > 2  # 0 is builtins, 1 globals, then local scope
            and hasattr(node.target, "id")
            and node.target.id not in self.scope.dicts[-1]
        ):
            raise UnboundLocalError(
                f"local variable '{node.target.id}' referenced before assignment"
            )
        try:
            value = self.operators[type(node.op)](
                self._eval(node.target), self._eval(node.value)
            )
        except KeyError:  # pragma: no cover
            # This is caught in validate()
            raise NotImplementedError(
                "Sorry, {0} is not available in this "
                "evaluator".format(type(node.op).__name__)
            )
        return self._assign([node.target], value)

    def _eval_assign(self, node):
        value = self._eval(node.value)
        return self._assign(node.targets, value)

    def _eval_annassign(self, node):
        if node.value:
            value = self._eval(node.value)
            return self._assign([node.target], value)

    def _eval_namedexpr(self, node):
        # NamedExpr(
        #     target=Name(id='x', ctx=Store()),
        #     value=Constant(value=4))
        value = self._eval(node.value)
        self._assign([node.target], value)
        return value

    @staticmethod
    def _eval_constant(node):
        if len(repr(node.value)) > MAX_STRING_LENGTH:
            raise MemoryError(
                "Value is too large ({0} > {1} )".format(
                    len(repr(node.value)), MAX_STRING_LENGTH
                )
            )
        return node.value

    @staticmethod
    def _eval_num(node):
        if len(repr(node.n)) > MAX_STRING_LENGTH:
            raise MemoryError(
                "Value is too large ({0} > {1} )".format(
                    len(repr(node.n)), MAX_STRING_LENGTH
                )
            )
        return node.n

    @staticmethod
    def _eval_bytes(node):
        if len(node.s) > MAX_STRING_LENGTH:
            raise MemoryError(
                "Byte Literal in statement is too long!"
                " ({0}, when {1} is max)".format(len(node.s), MAX_STRING_LENGTH)
            )
        return node.s

    @staticmethod
    def _eval_str(node):
        if len(node.s) > MAX_STRING_LENGTH:
            raise MemoryError(
                "String Literal in statement is too long!"
                " ({0}, when {1} is max)".format(len(node.s), MAX_STRING_LENGTH)
            )
        return node.s

    @staticmethod
    def _eval_nameconstant(node):
        return node.value

    def _eval_unaryop(self, node):
        return self.operators[type(node.op)](self._eval(node.operand))

    def _eval_binop(self, node):
        try:
            return self.operators[type(node.op)](
                self._eval(node.left), self._eval(node.right)
            )
        except KeyError:  # pragma: no cover
            # This is caught in validate()
            raise NotImplementedError(
                "Sorry, {0} is not available in this "
                "evaluator".format(type(node.op).__name__)
            )

    def _eval_boolop(self, node):
        if isinstance(node.op, ast.And):
            vout = False
            for value in node.values:
                vout = self._eval(value)
                if not vout:
                    return vout
            return vout
        elif isinstance(node.op, ast.Or):
            for value in node.values:
                vout = self._eval(value)
                if vout:
                    return vout
            return vout
        else:  # pragma: no cover
            # This should never happen as there are only two bool operators And and Or
            raise NotImplementedError(
                "Sorry, {0} is not available in this "
                "evaluator".format(type(node).__name__)
            )

    def _eval_compare(self, node):
        right = self._eval(node.left)
        to_return = True
        for operation, comp in zip(node.ops, node.comparators):
            if not to_return:
                break
            left = right
            right = self._eval(comp)
            to_return = self.operators[type(operation)](left, right)
        return to_return

    def _eval_ifexp(self, node):
        return (
            self._eval(node.body) if self._eval(node.test) else self._eval(node.orelse)
        )

    def _eval_if(self, node):
        if self._eval(node.test):
            [self._eval(b) for b in node.body]
        else:
            [self._eval(b) for b in node.orelse]

    def _eval_try(self, node):
        try:
            for b in node.body:
                self._eval(b)
        except:  # noqa: E722
            caught = False
            for h in node.handlers:
                if self._eval(h):
                    caught = True
                    break
            if not caught:
                raise
        else:
            [self._eval(oe) for oe in node.orelse]
        finally:
            [self._eval(f) for f in node.finalbody]

    def _eval_excepthandler(self, node):
        _type, exc, traceback = sys.exc_info()
        if isinstance(exc, Return):
            return False

        if isinstance(exc, UNCATCHABLE_EXCEPTIONS) or (
            isinstance(exc, DysRuntimeError)
            and isinstance(exc.__context__, UNCATCHABLE_EXCEPTIONS)
        ):
            return False
        if (
            (node.type is None)
            or isinstance(exc, self._eval(node.type))
            or (
                isinstance(exc, DysRuntimeError)
                and isinstance(exc.__context__, self._eval(node.type))
            )
        ):
            # Surprisingly this is how python does it
            # See: https://docs.python.org/3/reference/compound_stmts.html#the-try-statement
            if node.name:
                self.scope[node.name] = exc
            try:
                [self._eval(b) for b in node.body]
            finally:
                if node.name in self.scope:
                    del self.scope[node.name]
            return True
        return False

    def _eval_call(self, node):
        if len(self.call_stack) >= MAX_CALL_DEPTH:
            raise RecursionError("Sorry, stack is to large")

        func = self._eval(node.func)
        if not callable(func):
            raise TypeError(
                "Sorry, {} type is not callable".format(type(func).__name__)
            )

        assert_func_allowed(func)
        kwarg_kwargs = [self._eval(k) for k in node.keywords]

        for a in node.args:
            if a.__class__ == ast.Starred:
                args = self._eval(a.value)
            else:
                args = [self._eval(a)]
            func = partial(func, *args)
        for kwargs in kwarg_kwargs:
            func = partial(func, **kwargs)

        self.call_stack.append([node, self.expr])
        try:
            ret = func()
        except:
            raise
        finally:
            self.call_stack.pop()
        return ret

    def _eval_keyword(self, node):
        if node.arg is not None:
            return {node.arg: self._eval(node.value)}
        # Not possible until kwargs are enabled
        return self._eval(node.value)

    def _eval_name(self, node):
        try:
            return self.scope[node.id]
        except KeyError:
            msg = "'{0}' is not defined".format(node.id)
            raise NameError(msg)
            # raise NameNotDefined(node)

    def _eval_subscript(self, node):
        container = self._eval(node.value)
        key = self._eval(node.slice)
        return container[key]

    def _eval_attribute(self, node):
        for prefix in DISALLOW_PREFIXES:
            if node.attr.startswith(prefix):
                raise NotImplementedError(
                    "Sorry, access to this attribute "
                    "is not available. "
                    "({0})".format(node.attr)
                )
        # eval node
        node_evaluated = self._eval(node.value)
        if (type(node_evaluated), node.attr) in DISALLOW_METHODS:
            raise DangerousValue(
                "Sorry, this method is not available. "
                "({0}.{1})".format(node_evaluated.__class__.__name__, node.attr)
            )
        return getattr(node_evaluated, node.attr)

    def _eval_index(self, node):
        return self._eval(node.value)

    def _eval_slice(self, node):
        lower = upper = step = None
        if node.lower is not None:
            lower = self._eval(node.lower)
        if node.upper is not None:
            upper = self._eval(node.upper)
        if node.step is not None:
            step = self._eval(node.step)
        return slice(lower, upper, step)

    def _eval_joinedstr(self, node):
        length = 0
        evaluated_values = []
        for n in node.values:
            val = str(self._eval(n))
            if len(val) + length > MAX_STRING_LENGTH:
                raise MemoryError("Sorry, I will not evaluate something this long.")
            length += len(val)
            evaluated_values.append(val)
        return "".join(evaluated_values)

    def _eval_formattedvalue(self, node):
        if node.format_spec:
            # from https://stackoverflow.com/a/44553570/260366

            format_spec = self._eval(node.format_spec)
            r = r"(([\s\S])?([<>=\^]))?([\+\- ])?([#])?([0])?(\d*)([,])?((\.)(\d*))?([sbcdoxXneEfFgGn%])?"
            FormatSpec = nt(
                "FormatSpec",
                "fill align sign alt zero_padding width comma decimal precision type",
            )
            match = re.fullmatch(r, format_spec)

            if match:
                parsed_spec = FormatSpec(
                    *match.group(2, 3, 4, 5, 6, 7, 8, 10, 11, 12)
                )  # skip groups not interested in
                if int(parsed_spec.width or 0) > 100:
                    raise MemoryError("Sorry, this format width is too long.")

                if int(parsed_spec.precision or 0) > 100:
                    raise MemoryError("Sorry, this format precision is too long.")

            conversion_dict = {-1: "", 115: "!s", 114: "!r", 97: "!a"}

            fmt = "{" + conversion_dict[node.conversion] + ":" + format_spec + "}"
            return fmt.format(self._eval(node.value))
        return self._eval(node.value)

    def _eval_dict(self, node):
        if len(node.keys) > MAX_STRING_LENGTH:
            raise MemoryError("Dict in statement is too long!")
        res = {}
        for k, v in zip(node.keys, node.values):
            if k is None:
                res.update(self._eval(v))
            else:
                res[self._eval(k)] = self._eval(v)

        return res

    def _eval_tuple(self, node):
        if len(node.elts) > MAX_STRING_LENGTH:
            raise MemoryError("Tuple in statement is too long!")
        return tuple(self._eval(x) for x in node.elts)

    def _eval_list(self, node):
        if len(node.elts) > MAX_STRING_LENGTH:
            raise MemoryError("List in statement is too long!")
        return list(self._eval(x) for x in node.elts)

    def _eval_set(self, node):
        if len(node.elts) > MAX_STRING_LENGTH:
            raise MemoryError("Set in statement is too long!")
        return set(self._eval(x) for x in node.elts)

    def track(self, node):
        if hasattr(node, "nodes_called"):
            return

        self.nodes_called += 1
        if self.nodes_called > MAX_NODE_CALLS:
            raise TimeoutError("This program has too many evaluations")
        size = len(repr(self.scope)) + len(repr(self._last_eval_result))
        if size > MAX_SCOPE_SIZE:
            raise MemoryError("Scope has used too much memory")

    def _eval_comprehension(self, node):

        if isinstance(node, ast.ListComp):
            to_return = list()
        elif isinstance(node, ast.DictComp):
            to_return = dict()
        elif isinstance(node, ast.SetComp):
            to_return = set()
        else:  # pragma: no cover
            raise Exception("should never happen")

        self.scope.push({})

        def recurse_targets(target, value):
            """
                Recursively (enter, (into, (nested, name), unpacking)) = \
                             and, (assign, (values, to), each
            """
            self.track(target)
            if isinstance(target, ast.Name):
                self.scope[target.id] = value
            else:
                for t, v in zip(target.elts, value):
                    recurse_targets(t, v)

        def do_generator(gi=0):
            g = node.generators[gi]

            for i in self._eval(g.iter):
                recurse_targets(g.target, i)
                if all(self._eval(iff) for iff in g.ifs):
                    if len(node.generators) > gi + 1:
                        do_generator(gi + 1)
                    else:
                        if isinstance(node, ast.ListComp):
                            to_return.append(self._eval(node.elt))
                        elif isinstance(node, ast.DictComp):
                            to_return[self._eval(node.key)] = self._eval(node.value)
                        elif isinstance(node, ast.SetComp):
                            to_return.add(self._eval(node.elt))
                        else:  # pragma: no cover
                            raise Exception("should never happen")

        do_generator()

        self.scope.dicts.pop()
        return to_return


def dys_eval(expr, scope=None, call_stack=None, module_dict=None):
    """Simply evaluate an expresssion"""

    modules = None
    if module_dict:
        modules = make_modules(module_dict)

    s = DysEval(scope=scope, modules=modules, call_stack=call_stack)
    return s.eval(expr)


class DysCoverage(DysEval):

    seen_nodes = defaultdict(int)

    def __init__(self, *args, **kwargs):
        return super(DysCoverage, self).__init__(*args, **kwargs)

    def eval(self, expr):
        self.seen_nodes = {
            (
                n.lineno,
                n.col_offset,
                n.end_lineno,
                n.end_col_offset,
                n.__class__.__name__,
            ): 0
            for n in ast.walk(ast.parse(expr))
            if hasattr(n, "col_offset")
        }
        return super(DysCoverage, self).eval(expr)

    def _assign(self, targets, value):
        ret = super(DysCoverage, self)._assign(targets, value)
        # currently only one target is allowed, but still.
        for node in targets:
            self.track(node)
        return ret

    def _eval(self, node):
        ret = super(DysCoverage, self)._eval(node)
        self.track(node)
        return ret

    def _eval_arguments(self, node):
        ret = super(DysCoverage, self)._eval_arguments(node)
        for node_arg in node.args:
            self.track(node_arg)
        return ret

    def track(self, node):
        if hasattr(node, "seen_nodes"):
            xx = Counter(node.seen_nodes)
            yy = Counter(self.seen_nodes)
            xx.update(yy)
            self.seen_nodes = dict(xx)
        if hasattr(node, "col_offset"):
            self.seen_nodes[
                (
                    node.lineno,
                    node.col_offset,
                    node.end_lineno,
                    node.end_col_offset,
                    node.__class__.__name__,
                )
            ] += 1


def dys_test_coverage(expr, scope=None, call_stack=None, module_dict=None):
    """Run all test_* function in this expression"""

    modules = make_modules(module_dict or {})

    s = DysCoverage(scope=scope, modules=modules, call_stack=call_stack)
    s.eval(expr)
    test_names = [n for n in s.scope if n.startswith("test_") and callable(s.scope[n])]
    for name in test_names:
        s.scope[name]()
    return sorted(s.seen_nodes.items())


def ascii_format_coverage(coverage, source):
    pct = sum(v > 0 for k, v in coverage) / len(coverage)
    # total = sum(v for k, v in coverage)
    out = ""
    for (r, c, name), v in coverage:
        if v:
            continue
        out += f"Missing {name} on line: {r} col: {c}\n"
        out += (source.splitlines()[r - 1]) + "\n"
        out += (c * "-") + "^\n"
    out += f"{ int(pct * 100) }% coverage\n"
    return out
