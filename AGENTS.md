# For agents and contributors

##  Back-end development

* API development is spec-driven. We generate the server code from our openapi spec located in [./openapi](./openapi).
  * Code is generated using `go generate ./...`.
* The OpenAPI code generator creates an interface called `ServerInterface` which we implement in the various files of our [./api/server](./api/server) package (one file per REST resource).
* We use `goose` for database migrations (`go tool goose -h`).
* We use `sqlc` for database code generation (also handled by `go generate ./...`), it is configured to work with `goose`.

### Back-end code style

* If an error is returned by a function, we should always wrap it with more information: `return nil, err // bad`, `return nil, fmt.Errorf("the user could not be created: %w", err) // good`.
* We have a generic function `ptr.Ptr` to return a pointer from a value.

## Front-end development

* Code lives in [frontend/](./frontend/).
* We use pnpm, React, TypeScript, Vite, Tailwind and Shadcn.
* We generate the client code from the OpenAPI spec using `cd frontend; nvm use .; pnpm run openapi-ts`. The generated code is located in [frontend/src/opengym/client](./frontend/src/opengym/client).
* We import ui components from Shadcn using the CLI e.g. `pnpm dlx shadcn@latest add button card --overwrite --yes`.
* We use Ladle for component development (`pnpm ladle serve`).
* We use react-i18next for internationalization, the locales are located in [frontend/src/i18n/locales](./frontend/src/i18n/locales).
