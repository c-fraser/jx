# jx

[![Release](https://img.shields.io/github/v/release/c-fraser/jx?logo=github&sort=semver)](https://github.com/c-fraser/jx/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/c-fraser/jx)](https://goreportcard.com/report/github.com/c-fraser/jx)
[![Apache License 2.0](https://img.shields.io/badge/License-Apache2-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)

`jx` makes managing and executing JVM (CLI) applications effortless, similar to 
Python's [pipx](https://github.com/pypa/pipx/) and JavaScript's [npx](https://github.com/npm/npx).

## Install

### Homebrew

```shell
brew install c-fraser/tap/jx
```

### Go

```shell
go install github.com/c-fraser/jx
```

### Releases

Download a `jx` binary from a [release](https://github.com/c-fraser/jx/releases). 

## Usage

### Install a [Gradle](https://gradle.org/) project

```shell
jx install gradle --git git@github.com:c-fraser/echo.git
```

> If the project name is not specified then the repository name is used. 

### Run an installed project

```shell
jx run echo 'Hello, World!'
```

### Upgrade an installed project

```shell
jx upgrade echo
```

### Uninstall a project

```shell
jx uninstall echo
```

### View the documentation

```shell
jx --help
```

## License

    Copyright 2022 c-fraser
    
    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at
    
        https://www.apache.org/licenses/LICENSE-2.0
    
    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
