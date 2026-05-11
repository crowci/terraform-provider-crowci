# Terraform provider for Crow CI server

## Development

### Setting up test Crow CI with localhost forgejo instance

1. Install [mise](https://mise.jdx.dev/) and setup the required dependencies

   ```bash
   mise i
   ```

2. Setup test forgejo and crow instance

   ```bash
   docker compose up forgejo -d
   ```

3. Go to the test Forgejo instance at `http://localhost:3000` and create a test user
4. Go to `http://localhost:3000/user/settings/applications` and create an OAuth2 application
   - Application name: Crowci
   - Redirect URLs: `http://localhost:8000/authorize`
   - Confidential clients: Checked
5. Then paste the `Client ID` and `Client secret` into the `.env` file as of following:

   ```bash
   CROW_FORGEJO_CLIENT=<Client ID>
   CROW_FORGEJO_SECRET=<Client secret>
   ```

6. Setup the crow CI instance

   ```bash
   docker compose up -d
   ```

7. Login the Crow CI instance at `http://localhost:8000/user/access-tokens` and create an access tokens with all the permission
8. Put the token into the `.env` file

   ```bash
   CROWCI_TOKEN=<Access token>
   ```

9. Install pre-commit hook

   ```bash
   prek i
   ```

10. Check out the commands in [mise.toml](./mise.toml) file for available quick commands (.e.g: run test, generate documentations,...)

### Development notes

- Getting the crow API `openapi.json` file

   ```bash
   mkdir -p .local
   curl http://localhost:8000/api/v1/openapi.json --output ./.local/openapi.json
   ```

- This file can be useful for feeding into coding agent for generating the code
- Endpoint for api docs: `http://localhost:8000/api/v1/docs`
