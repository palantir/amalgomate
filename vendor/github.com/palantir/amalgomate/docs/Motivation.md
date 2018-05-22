There are many stand-alone Go programs that provide useful functionality to perform a certain task. One example of this is static analysis/linting of code -- there are many different programs like [`errcheck`](https://github.com/kisielk/errcheck), [`go vet`](https://golang.org/cmd/vet/) and [`deadcode`](https://github.com/tsenart/deadcode) that perform a specific portion of a useful task. These are all separate stand-alone Go programs, but they perform different aspects of a single larger task.

Because all of these programs are separate, they must be obtained separately. In order to perform all of the checks, both developers and our continuous deployment environment had to fetch all of these programs separately:

```
go get -u -v \
                github.com/golang/lint/golint \
                github.com/gordonklaus/ineffassign \
                github.com/jstemmer/go-junit-report \
                github.com/kisielk/errcheck \
                github.com/mdempsky/unconvert \
                github.com/opennota/check/cmd/varcheck \
                github.com/pierrre/gotestcover \
                github.com/remyoudompheng/go-misc/deadcode \
                golang.org/x/tools/cmd/cover
```

This is less than ideal for the following reasons:
* Each of these programs is a compiled stand-alone executable coming it at 5-10MB, so the total size of downloaded/built executables adds up rapidly (in the example above, it was around 90MB)
* The `go get` mechanism fetches the latest version of the project at the time it is run, so different developers/environments could end up with different versions of the checks
  * Although it is possible to work around this by specifying specific versions for the downloads, that is more complicated (it basically requires dependency management across the team)
* Some external script or program is required to invoke all of these different programs (unless they are all run manually)

There are projects like [gometalinter](https://github.com/alecthomas/gometalinter) that address the last portion of this problem, but it does not solve the other problems that are outlined (it provides a mechanism for installing other linters, but that mechanism still downloads each of them separately and results in independent executables).

The ideal would be to have a **single executable** that provides the functionality of all of the separate programs.