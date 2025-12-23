# For agents and contributors

##  Back-end development

* API development is spec-driven. We generate the server code from our openapi spec located in [./openapi](./openapi).
  * Code is generated using `go generate ./...`.
* The OpenAPI code generator creates an interface called `ServerInterface` which we implement in the various files of our [./api/server](./api/server) package (one file per REST resource).

### Back-end code style

* If an error is returned by a function, we should always wrap it with more information: `return nil, err // bad`, `return nil, fmt.Errorf("the user could not be created: %w", err) // good`.
