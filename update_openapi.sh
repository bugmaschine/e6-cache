echo "Deleting old routes"
rm ./src/openapi/e621.yaml

echo "Updating e621 routes"
curl -o ./src/openapi/e621.yaml https://raw.githubusercontent.com/DonovanDMC/E621OpenAPI/master/openapi.yaml
