# Ziglu RSS Reader
This project is a simple RSS feed service which consumes a list of feeds 
and then exposes their content via an easy to use API.

## Running
You can run the API server, tests, and Swagger OpenAPI documentation server
using `docker-compose`.

### API Server
Web server will be available on `localhost:8080`
```
docker-compose up reader -d
```

### Run tests
```
docker-compose run tests
```

### Documentation server
API documentation will be available on `localhost:8081`
```
docker-compose up docs -d
```