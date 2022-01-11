FROM registry.access.redhat.com/ubi8/nodejs-14:1-53 as build

WORKDIR /opt/app-root/src/marketplace-server

COPY package*.json ./

RUN npm install

COPY server.js .

EXPOSE 3000

CMD ["npm", "start"]
