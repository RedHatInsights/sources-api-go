// More info about this code in https://github.com/typicode/json-server
const jsonServer = require('json-server');
const server = jsonServer.create();
const middlewares = jsonServer.defaults();

server.use(middlewares);

// Whenever a request is received, return a fake token with an expiration date on 2050-01-01T00:00:00
// https://marketplace.redhat.com/en-us/documentation/api-authentication
server.post('/api-security/om-auth/cloud/token', (req, res) => {
    res.json(
        {
            access_token: "fakeString",
            expiration: 2524604400
        }
    )
});

server.listen(3000);
