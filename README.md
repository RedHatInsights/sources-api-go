# --== Sources API ==--
###### _but in Go_

Here lies the source code for the Sources API re-write, based on the [original Rails Application](https://github.com/RedHatInsights/sources-api)

### Quick Info
- Written in Go, using [echo](https://echo.labstack.com/) to handle HTTP routing layer + middleware
- using [GORM](gorm.io) for the DAO layer and interacting with the database
- clowder enabled

### Repo Layout
|Folder/File   | Contents  |
|---|---|
| `config/` | configuration file reading Clowder cdappconfig.json \|\| ENV vars |
| `middleware/` | middleware functions for parsing headers, validating account numbers, etc |
| `model/` | structs representing db records + http requests for each model |
| `dao/`  |  structs with the methods for interacting with the database (e.g. list all, get by id, update, etc) |
| `util/` | misc for responses, etc |
| `redis/` | redis client |
| `*_handlers.go` | http handlers for the app, usually just parses requests into models, reaches into DAO, then returns response. |
| `routes.go` | contains ALL THE ROUTES! so it's easy to view the mounted routes + what middleware is being used per route |

### Development
- Check out the repository, then run `make setup` to download the dependencies
- The `Makefile` contains various targets for development, e.g.
    - `make run` to build the binary + run
    - `make inlinerun` to just run the application inline (no output binary, all in memory)
    - `make debug` to run `dlv debug`, allowing setting of breakpoints etc
    - `make tidy` to check go files for new imports and add them to `go.sum`
    - `make lint` to run the same linters as the PR action, and print errors.
- Tests are currently in the same package adjacent to the source file. ex: `source_handlers.go` -> `source_handlers_test.go`, just using the standard library testing library. May change in the future.

### Launch the integration tests

In order to launch the integration tests just do the following:

- Launch a test database with the following command: `podman run --rm --name sources_test -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=sources_api_test_go -p 5432:5432 --rm docker.io/postgres:13 postgres -c log_statement=all`
- Launch the tests themselves: `DATABASE_HOST=localhost DATABASE_PORT=5432 DATABASE_USER=postgres DATABASE_PASSWORD=postgres make alltest`

## Licenses

This project is available as open source under the terms of the [Apache License 2.0](http://www.apache.org/licenses/LICENSE-2.0).
