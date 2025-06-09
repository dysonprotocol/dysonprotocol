if __name__ == "__main__":
    import sys

    if sys.argv[1] == "exec_script":
        from . import dysvm_server
        dysvm_server.main(*sys.argv[2:])

    elif sys.argv[1] == "run_wsgi":
        from . import dyswsgi
        dyswsgi.main(*sys.argv[2:])
    
    elif sys.argv[1] == "dys_format":
        import black
        code = sys.stdin.read()
        try:
            formatted_code = black.format_str(code, mode=black.Mode())
            #import ast
            #formatted_code = ast.unparse(ast.parse(code))
            print(formatted_code)
        except Exception as e:
            print(f"Error formatting code: {e}", file=sys.stderr)
            sys.exit(1)

