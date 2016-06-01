FROM mhart/alpine-node
MAINTAINER Eugene Brodsky <boomstik@gmail.com>
WORKDIR /opt/
COPY . .
RUN npm install
ENTRYPOINT ["node", "crash.js"]