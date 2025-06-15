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
# clean up old temp files
rm -rf ./client/docs/swagger-ui/swagger-gen || true


###
# Generate the swagger docs
###


mkdir -p ./client/docs/swagger-ui/swagger-gen

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
pwd

# Python function to remove specific description keys 
python - <<EOF
print("Starting python script")
import json
import glob

def walk_and_modify(obj):
    if isinstance(obj, dict):
        for key in list(obj.keys()):
            if key == 'description' and isinstance(obj[key], str) and (obj[key].startswith('\`Any\`') or obj[key].startswith('A URL/resource')):
                print(f"Removing description")
                del obj[key]
            else:
                walk_and_modify(obj[key])
    elif isinstance(obj, list):
        for item in obj:
            walk_and_modify(item)

def remove_specific_descriptions(json_file):
    with open(json_file, 'r') as file:
        data = json.load(file)

    walk_and_modify(data)

    with open(json_file, 'w') as file:
        json.dump(data, file, indent=2)
        file.flush()

for file in glob.glob('./client/docs/swagger-ui/swagger-gen/**/*.json',recursive=True):
    print(f"remove_specific_descriptions(file={file})")
    remove_specific_descriptions(file)
EOF

# combine swagger files
# uses nodejs package `swagger-combine`.
# all the individual swagger files need to be configured in `config.json` for merging
swagger-combine ./client/docs/config.json -o ./client/docs/swagger-ui/swagger.yaml -f yaml --continueOnConflictingPaths true --includeDefinitions true


cd ./client/docs/swagger-ui/swagger-gen
tree * -J > index.json
tree * --prune -H "./" > index.html
cd -


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



echo "Proto generation complete!" 
