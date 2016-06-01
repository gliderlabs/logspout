// 'use strict'
// http://expressjs.com/en/starter/hello-world.html
let express = require('express'),
  uuid = require('uuid');

let app = express();

app.get('/', function (req, res) {
  let date = new Date().toJSON();
  res.send(date);
  // var date = new Date().toJSON();
  console.log(date, '\t', uuid.v4());
});

app.listen(3000, function () {
  let date = new Date().toJSON();
  console.log(date +'\tBEGIN\n--------------------------------------------------------');
});