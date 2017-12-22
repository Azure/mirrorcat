
<p align="center"><img src="https://github.com/ashleymcnamara/gophers/blob/53a51e151e368eb57ef5958588365f6e3a6cd6e2/MovingGopher.png" width="360">
</p>
<p align="center">
    <a href="https://travis-ci.org/Azure/mirrorcat"><img src="https://travis-ci.org/Azure/mirrorcat.svg?branch=master"></a>
    <a href="https://godoc.org/github.com/Azure/mirrorcat"><img src="https://godoc.org/github.com/Azure/mirrorcat?status.svg" alt="GoDoc"></a>
</p>


# MirrorCat

Tired of manually keeping branches up-to-date with one another across repositories? Are [Git Hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks) not enough for you for some reason? Deploy your instance of MirrorCat to expose a web service which will push your commits around to where they are needed.

## Acquire

### Docker

You can run MirrorCat in a container, however/where ever you want, using the Docker. You can find the most up-to-date images here:
https://cloud.docker.com/swarm/marstr/repository/docker/marstr/mirrorcat/general

### Built from Source

__Note:__ To do this, you'll need the most recent version of the Go Programming Language installed on your machine. It also helps to have [`dep`](https://github.com/golang/dep) installed.

The simplest, albeit least stable, way to install mirrorcat is to use `go get` as seen below:

``` bash
go get -u github.com/Azure/mirrorcat/mirrorcat
```

A somewhat more stable way to install is to separately fetch the source and build it yourself, as below:

``` bash
go get -d github.com/Azure/mirrorcat
cd $GOPATH/src/github.com/Azure/mirrorcat
dep ensure
go install -ldflags "-X github.com/Azure/mirrorcat/mirrorcat/cmd.commit=$(git rev-parse HEAD)" ./mirrorcat
```
or if you're a PowerShell-type-person:
``` PowerShell
go get -d github.com/Azure/mirrorcat
cd $env:GOPATH\src\github.com\Azure\mirrorcat
dep ensure
go install -ldflags "-X github.com/Azure/mirrorcat/mirrorcat/cmd.commit=$(git rev-parse HEAD)" .\mirrorcat
```

## Configure

You'll need to tell MirrorCat which branches to mirror, and where. There are a couple of ways of doing that. The best part? You don't even need to choose between them. A MirrorCat looks around when it starts up to see what is available to it.

### Available Options

| Flag               | Config File Key  | Environment Variable       | Default          | Usage                                                                                      |
| :----------------: | :--------------: | :------------------------: | :--------------: | ------------------------------------------------------------------------------------------ |
| --config           | N/A              | MIRRORCAT_CONFIG           | ~/.mirrorcat.yml | The configuration file that should be used for all of the following settings.              |
| --port             | port             | MIRRORCAT_PORT             | 8080             | The TCP port that should be used to serve this instance of MirrorCat                       |
| --redis-connection | redis-connection | MIRRORCAT_REDIS_CONNECTION | _None_           | The connection string MirrorCat to use while looking for branch mappings in a Redis cache. |
| --clone-depth      | clone-depth      | MIRRORCAT_CLONE_DEPTH      | _Infinity_       | The number of commits that should be cloned while moving commits between repositories.     |
| N/A                | mirrors          | N/A                        | _None_           | A mapping of which branches are to be copied from one repository to another.               |

### Using a Config File

In addition to specifying administrative stuff, you can provide lists of where to copy each branch using either JSON or YAML. MirrorCat watches the file that was used as start-up, if the file changes MirrorCat will attempt to reconfigure itself while running. 

#### .mirrorcat.yml
``` yaml
port: 8080
redis-connection: redis://localhost:6379/0
mirrors:
  https://github.com/Azure/mirrorcat.git:
    master:
      https://github.com/marstr/mirrorcat.git:
      - master
  https://github.com/Azure/azure-sdk-for-go.git:
    master:
      https://github.com/Azure/azure-sdk-for-go.git:
      - dev
```
#### .mirrorcat.json
``` json
{
  "port": 8080,
  "redis-connection": "redis://localhost:6379/0",
  "mirrors": {
    "https://github.com/Azure/mirrorcat.git": {
      "master": {
        "https://github.com/marstr/mirrorcat.git": [
          "master"
        ]
      }
    },
    "https://github.com/Azure/azure-sdk-for-go.git": {
      "master": {
        "https://github.com/Azure/azure-sdk-for-go.git": [
          "dev"
        ]
      }
    }
  }
}
```

### Using Redis

Sometimes, you may want to introduce some dynamicism into how MirrorCat behaves. For example, you may want to have a website where users can declare a branch they've been working on in a lieutenant repository ready for the big time. [Redis is a great way to enable this](https://redis.io/). Just point MirrorCat at a Redis instance by passing it a Redis connection string.

The expected schema of the Redis instance is to have [Sets](https://redis.io/commands/sadd) of mappings between branch/repository pairs separated by the rune ':'. 

For example, a Redis instance where the following commands had been run would result in MirrorCat behaving the same as a MirrorCat using the config files from above.

``` redis
SADD master:https://github.com/Azure/mirrorcat.git master:https://github.com/marstr/mirrorcat.git
SADD master:https://github.com/Azure/azure-sdk-for-go.git dev:https://github.com/Azure/azure-sdk-for-go.git
```

### Using Environment Variables

Any flag that you can provide the `mirrorcat start` command can be provided in the config file mentioned above. However, you can also speficy it by setting an environment variable prefixed with the name "mirrorcat" and all hyphensreplaced with underscores ('-' -> '_').

### Priority

MirrorCat, being a project written in Go, uses the really awesome libraries `github.com/spf13/cobra` and `github.com/spf13/viper` to easily deal with flags and configuration. As such, MirrorCat uses the following priority list when looking at the configuration handed to it:

1. Command Line Arguments
1. Environment Variables
1. Configuration File
1. Default Values

## Contribute

### Conduct
If you would like to become an active contributor to this project please follow the instructions provided in Microsoft Azure Projects Contribution Guidelines.

This project has adopted the Microsoft Open Source Code of Conduct. For more information see the Code of Conduct FAQ or contact opencode@microsoft.com with any additional questions or comments.

### Requirements

You'll need the following tools to build and test MirrorCat:

- [The Go Programming Language, 1.9 or later.](https://golang.org/dl/)
- [`dep`](https://github.com/golang/dep)
- [`git`](https://git-scm.org)

The easiest way to get a hold of MirrorCat's source is using `go get`:

``` bash
go get -d -t github.com/Azure/mirrorcat
```

### Running Tests

Once you've acquired the source, you can run MirrorCat's tests with the following command:

``` bash
go test -race -cover -v github.com/Azure/mirrorcat/...
```

## Attribution

The gopher image at the top of this README was created by Ashley McNamara. You can find this image, and many others like it, at https://github.com/ashleymcnamara/gophers
