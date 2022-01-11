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

#### Third party services

In order to run the services this application will connect to, a `docker-compose.yml` file is provided. This will
launch the Redis cache, the database and a Vault server in development mode. The steps to have a working development
environment are:

1. `docker-compose up`
2. Go to the `sources-api` repository and migrate the database.
   1. `bundle exec rake db:migrate`
   2. `bundle exec rake db:seed`
3. Export the minimum required environment variables. You can modify some of them by modifying the `.env` file:

```bash
#!/usr/bin/bash

export BYPASS_RBAC=true
export DATABASE_HOST=localhost
export DATABASE_NAME=sources_api_development
export DATABASE_PASSWORD=toor
export DATABASE_PORT=5432
export DATABASE_USER=root
export REDIS_CACHE_HOST=localhost
export REDIS_CACHE_PORT=6379
export VAULT_ADDR=http://0.0.0.0:8200
export VAULT_TOKEN=root
```
