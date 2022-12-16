// Copyright 2022 c-fraser
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	repository = "git@github.com:c-fraser/echo.git"
	name       = "echo"
)

func TestCLI(t *testing.T) {
	if _, code := jx(t, "run", name); code != 1 {
		t.Fatal("(uninstalled project) ran successfully")
	}
	if _, code := jx(
		t,
		"install",
		"--git",
		repository,
		"--name",
		name,
	); code != 0 {
		t.Fatal("install failed")
	}
	if output, code := jx(t, "run", name, "test"); output != "test" && code != 0 {
		t.Logf("output: %s, exit code: %v", output, code)
		t.Fatal("run failed")
	}
	if _, code := jx(t, "uninstall", name); code != 0 {
		t.Fatal("uninstall failed")
	}
	if _, code := jx(t, "run", name); code != 1 {
		t.Fatal("(uninstalled project) ran successfully")
	}
}

func TestCommands(t *testing.T) {
	var conf config
	directory := t.TempDir()
	conf.File = filepath.Join(directory, "config.json")
	conf.Projects = make(map[string]project)
	if len(conf.Projects) != 0 {
		t.Errorf("%s is not empty", conf.Projects)
	}
	err := install(&conf, repository, filepath.Join(directory, name), name, "", "")
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if len(conf.Projects) != 1 {
		t.Errorf("%s is unexpected", conf.Projects)
	}
	proj, ok := conf.Projects[name]
	if !ok {
		t.Fatalf("%s not found in %s", name, conf.Projects)
	}
	err = run(&conf, name)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	err = update(proj.Repository)
	if err != nil {
		t.Fatalf("failed to update repository: %v", err)
	}
	err = upgrade(&conf, name)
	if err != nil {
		t.Fatalf("upgrade failed: %v", err)
	}
	err = run(&conf, name)
	if err != nil {
		t.Fatalf("run failed: %v", err)
	}
	err = uninstall(&conf, name)
	if err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}
	if len(conf.Projects) != 0 {
		t.Errorf("%s is not empty", conf.Projects)
	}
}

// jx runs the CLI command and returns the output and exit code.
func jx(t *testing.T, cli ...string) (string, int) {
	_, file, _, _ := runtime.Caller(0)
	current := filepath.Dir(file)
	var arguments []string
	command := "go"
	if runtime.GOOS == "windows" {
		command = command + ".exe"
	}
	arguments = append(arguments, "run", "main.go")
	arguments = append(arguments, cli...)
	cmd := exec.Command(command, arguments...)
	cmd.Dir = current
	output, err := cmd.Output()
	if err != nil {
		if process, ok := err.(*exec.ExitError); ok {
			return "", process.ExitCode()
		} else {
			t.Fatal("unexpected command error", err)
		}
	}
	return string(output), 0
}

// update (push a commit) the repository to verify upgrading an installed project.
func update(directory string) error {
	r, err := open(directory)
	if err != nil {
		return err
	}
	err = r.Fetch(&git.FetchOptions{})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	err = w.Pull(&git.PullOptions{})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return err
	}
	properties := filepath.Join(directory, "gradle.properties")
	now := strconv.FormatInt(time.Now().Unix(), 10)
	err = os.WriteFile(properties, []byte("version="+now), 0o644)
	if err != nil {
		return err
	}
	_, err = w.Add(filepath.Base(properties))
	if err != nil {
		return err
	}
	_, err = w.Commit(
		"testing "+now,
		&git.CommitOptions{
			Author: &object.Signature{
				Name:  "Chris Fraser",
				Email: "cfraser888@gmail.com",
				When:  time.Now(),
			},
		},
	)
	if err != nil {
		return err
	}
	err = r.Push(&git.PushOptions{})
	if err != nil {
		return err
	}
	return nil
}
