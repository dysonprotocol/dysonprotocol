#!/usr/bin/env bash

set -e
set -x

echo "Formatting protobuf files"
find ./proto -name "*.proto" -exec clang-format -i {} \;

echo "Deleting old generated files"
find ./api -name "*.pulsar.go" -delete
find ./api -name "*.pb.go" -delete
find ./x -name "*.pb.go" -delete
find ./x -name "*.pb.gw.go" -delete
# Also clean up old JSON schema files
rm -rf ./proto_json_schema

# clean up old temp files
rm -rf ./swagger-gen

###
# Generate go proto code for dysonprotocol modules
###


echo "Generating gogo proto code for dysonprotocol modules"
dyson_proto_dirs=$(find ./proto/dysonprotocol -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $dyson_proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    buf generate --template ./proto/buf.gen.gogo.yaml $file
    buf generate --template ./proto/buf.gen.pulsar.yaml $file
  done
done


# move proto files to the right places
cp -r ./dysonprotocol.com/* ./
rm -rf ./dysonprotocol.com

cp -r ./api/dysonprotocol.com/x/* ./api/
rm -rf ./api/dysonprotocol.com

###
# Generate JSON schemas for all proto files
###


# Ensure the JSON schema directory exists
mkdir -p ./proto_json_schema


echo "Generating JSON schemas for all proto files"
all_proto_files=$(find ./proto  \( -name 'query.proto' -o -name 'service.proto' -o -name 'tx.proto' \) \
  -not -path './proto/cosmos/app/*' \
  -not -path './proto/cosmos/autocli/*' \
  -not -path './proto/cosmos/base/*' \
  -not -path './proto/cosmos/benchmark/*' \
  -not -path './proto/cosmos/crypto/*' \
  -not -path './proto/cosmos/tx/*' \
  -not -path './proto/cosmos/reflection/*' \
  ) 
for file in $all_proto_files; do
  echo "Generating JSON schema for $file"
  if [ -f "$file" ]; then
    buf generate --template ./proto/buf.gen.jsonschema.yaml $file 
    
    # move the generated json schema from ./proto_json_schema to ./proto_json_schema/{basename of the file}
    original_path=$(dirname "$file")
    
    # remove the leading ./proto/
    original_path=${original_path#./proto/}
    # replace all slashes with dots
    new_path=${original_path//\//.}
    
    # get the basename of the proto file without the extension
    original_type=$(basename "$file" .proto)
    

    new_json_files=$(ls ./proto_json_schema/*.json || true)
    if [ -z "$new_json_files" ]; then
      echo "No JSON files found for $original_type"
      continue
    fi

    # if the directory does not exist, create it
    mkdir -p "./proto_json_schema/$new_path"
    
    # append the original_name without the extension to the json file names Foo.json -> Foo.tx.json    
    for json_file in $new_json_files; do
      new_name=$(basename "$json_file" .json)
      mv "$json_file" "./proto_json_schema/$new_path/$new_name.$original_type.json"

    done
    rm -rf ./proto_json_schema/*.json

  fi
done

# generate the index.html and index.json files
cd ./proto_json_schema
tree * --prune -H "./" > index.html
tree * -J > index.json
cd ..


###
# Generate the swagger docs
###


mkdir -p ./swagger-gen

cd ./proto
for d in $(find . -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq); do
  # skip if $dir is a subdirectory of ./cosmos/app/v1alpha1
  if [[ "$d" =~ ^./cosmos/app/v1alpha1 ]]; then
    echo "Skipping $d (subdirectory of ./cosmos/app/v1alpha1)"
    continue
  fi
  echo "Generating swagger files for $d"

  # only generate swagger files for tx.proto, query.proto, and service.proto
  proto_files=$(find "${d}" -maxdepth 1 \( -name 'tx.proto' -o -name 'query.proto' -o -name 'service.proto' \))
  for file in $proto_files; do
    buf generate --template buf.gen.swagger.yaml $file
  done

done

cd ..
# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine ./client/docs/config.json -o ./client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true

# Python function to remove specific description keys from the YAML
python - <<EOF
import yaml

def remove_specific_descriptions(yaml_file):
    with open(yaml_file, 'r') as file:
        data = yaml.safe_load(file)

    def walk_and_modify(obj):
        if isinstance(obj, dict):
            keys_to_delete = []
            for key, value in obj.items():
                if key == 'description' and isinstance(value, str) and (value.startswith('\`Any\`') or value.startswith('A URL/resource')):
                    keys_to_delete.append(key)
                else:
                    walk_and_modify(value)
            for key in keys_to_delete:
                del obj[key]
        elif isinstance(obj, list):
            for item in obj:
                walk_and_modify(item)

    walk_and_modify(data)

    with open(yaml_file, 'w') as file:
        yaml.safe_dump(data, file)

remove_specific_descriptions('./client/docs/swagger-ui/swagger.yaml')
EOF

echo "Proto generation complete!" 
