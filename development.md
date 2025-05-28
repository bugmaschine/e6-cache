
## Development
`e6-cache` works by forwarding the users request, which means that it should never require any client changes, and should work on anything that abides by the offical api.

## Request Forwarding Process
The core concept of the implementation looks like this:

1. Receive an api request
2. Forward the request to the target e6-based service (While checking for the Proxy Auth for example)
3. Capture the response
4. Modify the response and save it in the DB (URIs dont change in the DB)
5. Return the modified response to the client

## File Proxying Process
File Proxying works like this:

1. Check the Signature and decode the base64 encrypted url
2. Check in S3 if the file exists
3. If not, then request it and save it while forwarding it to the client. If it exist than stream it to the client from S3.

## OpenAPI Updates
The `update_openapi.sh` script:
- Updates the openai.yaml file from another repo