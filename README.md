# MyShell

MyShell is a customizable shell written in Go.

## Features

## Features

- **Job Management**: Start and manage jobs in the foreground and background.
- **Command History**: Track and recall command history.
- **Environment Variables**: Set and use environment variables.
- **Customizable Prompts**: Configure shell prompts and history settings.
- **Signal Handling**: Handle common Unix signals like SIGINT and SIGTSTP.

### Installation

1. **Clone the repository:**

   ```sh
   git clone https://github.com/sdfpt05/shell.git
   cd myshell
   ```

2. **Build the application:**

```sh
go build -o myshell ./cmd/shell
```

3. **Run the application:**

```sh
./shell
```

### Using Docker

1. Build the Docker image:

```sh
docker build -t shell .
```

2. **\*Run the Docker container:**

```sh
docker run -it myshell
```

3. **Configuration**

Configuration is managed via a YAML file. Example configuration:

```yaml
history_file: "/path/to/history_file"
home_dir: "/path/to/home_dir"
```

Save your configuration as config.yaml and adjust paths as needed. The default configuration will use the user's home directory and .shell_history file in it.

This is one of the John Cricket's Coding Challenges solutions https://codingchallenges.fyi/challenges/challenge-shell/
