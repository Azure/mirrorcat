This image was created by Ashley McNamara. You can find this image, and many others like it, at https://github.com/ashleymcnamara/gophers
<p align="center"><img src="https://github.com/ashleymcnamara/gophers/blob/53a51e151e368eb57ef5958588365f6e3a6cd6e2/MovingGopher.png" width="360"></p>
<p align="center">
    <a href="https://travis-ci.org/Azure/mirrorcat"><img src="https://travis-ci.org/Azure/mirrorcat.svg?branch=master"></a>
    <a href="https://godoc.org/github.com/Azure/mirrorcat"><img src="https://godoc.org/github.com/Azure/mirrorcat?status.svg" alt="GoDoc"></a>
</p>


# MirrorCat

Tired of manually keeping branches up-to-date with one another across repositories? Are [Git Hooks](https://git-scm.com/book/en/v2/Customizing-Git-Git-Hooks) not enough for you for some reason? Deploy your instance of MirrorCat to expose a web service which will push your commits around to where they are needed.

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
