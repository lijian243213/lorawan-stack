// Copyright © 2019 The Things Network Foundation, The Things Industries B.V.
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

package ttnmage

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Dev namespace.
type Dev mg.Namespace

// Misspell fixes common spelling mistakes in files.
func (Dev) Misspell() error {
	if mg.Verbose() {
		fmt.Println("Fixing common spelling mistakes in files")
	}
	return runGoTool("github.com/client9/misspell/cmd/misspell", "-w", "-i", "mosquitto",
		".editorconfig",
		".gitignore",
		".goreleaser.yml",
		".revive.toml",
		".travis.yml",
		"api",
		"cmd",
		"config",
		"CONTRIBUTING.md",
		"DEVELOPMENT.md",
		"doc",
		"docker-compose.yml",
		"Dockerfile",
		"lorawan-stack.go",
		"Makefile",
		"pkg",
		"README.md",
		"sdk",
		"SECURITY.md",
		"tools",
	)
}

var (
	sqlDatabase           = "cockroach"
	redisDatabase         = "redis"
	devDataDir            = ".env/data"
	devDatabaseName       = "ttn_lorawan_dev"
	devDockerComposeFlags = []string{"-p", "lorawan-stack-dev"}
	databaseURI           = fmt.Sprintf("postgresql://root@localhost:26257/%s?sslmode=disable", devDatabaseName)
)

var devDatabases = []string{sqlDatabase, redisDatabase}

func dockerComposeFlags(args ...string) []string {
	return append(devDockerComposeFlags, args...)
}

func execDockerCompose(args ...string) error {
	_, err := sh.Exec(nil, os.Stdout, os.Stderr, "docker-compose", dockerComposeFlags(args...)...)
	return err
}

func execDockerComposeWithOutput(filepath string, args ...string) error {
	output, err := sh.Output("docker-compose", dockerComposeFlags(args...)...)
	if err != nil {
		return err
	}
	message := []byte(output)
	err = ioutil.WriteFile(filepath, message, 0644)
	return err
}

// SQLStart starts the SQL database of the development environment.
func (Dev) SQLStart() error {
	if mg.Verbose() {
		fmt.Println("Starting SQL database")
	}
	if err := execDockerCompose(append([]string{"up", "-d"}, sqlDatabase)...); err != nil {
		return err
	}
	return execDockerCompose("ps")
}

// SQLStop stops the SQL database of the development environment.
func (Dev) SQLStop() error {
	if mg.Verbose() {
		fmt.Println("Stopping SQL database")
	}
	return execDockerCompose(append([]string{"stop"}, sqlDatabase)...)
}

// SQLMakeSnapshot stores the current cockroach data folder for later restores.
func (Dev) SQLMakeSnapshot() error {
	if mg.Verbose() {
		fmt.Println("Making DB snapshot")
	}
	to := filepath.Join(devDataDir, "cockroach-snap")
	from := filepath.Join(devDataDir, "cockroach")
	if err := os.RemoveAll(to); err != nil {
		return err
	}
	return sh.Copy(from, to)
}

// SQLRestoreSnapshot restores the previously taken snapshot, thus restoring all previously
// snapshoted databases.
func (d Dev) SQLRestoreSnapshot() error {
	mg.Deps(Dev.SQLStop)
	if mg.Verbose() {
		fmt.Println("Restoring DB snapshot")
	}
	to := filepath.Join(devDataDir, "cockroach")
	from := filepath.Join(devDataDir, "cockroach-snap")
	if err := os.RemoveAll(to); err != nil {
		return err
	}
	if err := sh.Copy(from, to); err != nil {
		return err
	}
	return d.SQLStart()
}

// DBDump performs an SQL database dump of the dev database to the .cache folder.
func (Dev) DBDump() error {
	if mg.Verbose() {
		fmt.Println("Saving database dump")
	}
	if err := os.MkdirAll(".cache", 0755); err != nil {
		return err
	}
	return execDockerComposeWithOutput(filepath.Join(".cache", "sqldump.sql"), "exec", "-T", "cockroach", "./cockroach", "dump", devDatabaseName, "--insecure")
}

// DBRestore restores the dev database using a previously generated dump.
func (Dev) DBRestore() error {
	mg.Deps(Js.Deps)
	if mg.Verbose() {
		fmt.Println("Restoring database from dump")
	}
	return sh.Run("node", filepath.Join("tools", "mage", "scripts", "restore-db-dump.js"), "--db", devDatabaseName, "--dump", filepath.Join(".cache", "sqldump.sql"))
}

// DBStart starts the databases of the development environment.
func (Dev) DBStart() error {
	if mg.Verbose() {
		fmt.Println("Starting dev databases")
	}
	if err := execDockerCompose(append([]string{"up", "-d"}, devDatabases...)...); err != nil {
		return err
	}
	return execDockerCompose("ps")
}

// DBStop stops the databases of the development environment.
func (Dev) DBStop() error {
	if mg.Verbose() {
		fmt.Println("Stopping dev databases")
	}
	return execDockerCompose(append([]string{"stop"}, devDatabases...)...)
}

// DBErase erases the databases of the development environment.
func (Dev) DBErase() error {
	mg.Deps(Dev.DBStop)
	if mg.Verbose() {
		fmt.Println("Erasing dev databases")
	}
	return os.RemoveAll(devDataDir)
}

// DBSQL starts an SQL shell.
func (Dev) DBSQL() error {
	mg.Deps(Dev.DBStart)
	if mg.Verbose() {
		fmt.Println("Starting SQL shell")
	}
	return execDockerCompose("exec", "cockroach", "./cockroach", "sql", "--insecure", "-d", devDatabaseName)
}

// DBRedisCli starts a Redis-CLI shell.
func (Dev) DBRedisCli() error {
	mg.Deps(Dev.DBStart)
	if mg.Verbose() {
		fmt.Println("Starting Redis-CLI shell")
	}
	return execDockerCompose("exec", "redis", "redis-cli")
}

// InitStack initializes the Stack.
func (Dev) InitStack() error {
	if mg.Verbose() {
		fmt.Println("Initializing the Stack")
	}
	if err := runGo("./cmd/ttn-lw-stack", "is-db", "init"); err != nil {
		return err
	}
	if err := runGo("./cmd/ttn-lw-stack", "is-db", "create-admin-user",
		"--id", "admin",
		"--email", "admin@localhost",
		"--password", "admin",
	); err != nil {
		return err
	}
	if err := runGo("./cmd/ttn-lw-stack", "is-db", "create-oauth-client",
		"--id", "cli",
		"--name", "Command Line Interface",
		"--owner", "admin",
		"--no-secret",
		"--redirect-uri", "local-callback",
		"--redirect-uri", "code",
	); err != nil {
		return err
	}
	return runGo("./cmd/ttn-lw-stack", "is-db", "create-oauth-client",
		"--id", "console",
		"--name", "Console",
		"--owner", "admin",
		"--secret", "console",
		"--redirect-uri", "https://localhost:8885/console/oauth/callback",
		"--redirect-uri", "http://localhost:1885/console/oauth/callback",
		"--redirect-uri", "/console/oauth/callback",
		"--logout-redirect-uri", "https://localhost:8885/console",
		"--logout-redirect-uri", "http://localhost:1885/console",
		"--logout-redirect-uri", "/console",
	)
}

// StartDevStack starts TTS in end-to-end test configuration.
func (Dev) StartDevStack() error {
	os.Setenv("TTN_LW_IS_DATABASE_URI", databaseURI)
	if mg.Verbose() {
		fmt.Println("Starting the Stack")
	}
	return runGoMute("./cmd/ttn-lw-stack", "start")
}

func init() {
	initDeps = append(initDeps, Dev.Certificates)
}
