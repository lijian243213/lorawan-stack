// Copyright © 2020 The Things Network Foundation, The Things Industries B.V.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

/* eslint-disable import/no-commonjs */

const fs = require('fs')

const { Client } = require('pg')
const yargs = require('yargs')

const argv = yargs.argv

const dbName = argv.db || 'ttn_lorawan_dev'
const dbDump = argv.dump || '.cache/sqldump.sql'

const sqlDump = fs.readFileSync(dbDump, 'utf8')

const client = new Client({
  user: 'root',
  host: 'localhost',
  database: dbName,
  port: 26257,
})
client.connect()

client.query(`DROP DATABASE ${dbName}`, err => {
  if (err) {
    throw err
  }

  client.query(`CREATE DATABASE ${dbName}`, err => {
    if (err) {
      throw err
    }

    client.query(sqlDump, err => {
      if (err) {
        throw err
      }

      client.end()
    })
  })
})
