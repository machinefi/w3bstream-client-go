# w3bstream-client-go

The Go Client for W3bstream integration on server

## Getting started

Install `w3bstream-client-go` 
``` shell
go get "github.com/machinefi/w3bstream-client-go"
```


## Example Code

### Publish Event Synchronously

``` go
import (
	"github.com/machinefi/w3bstream-client-go/client"
)

// the http_route, project and api_key are obtained on W3bstream-Studio
cli := NewClient("http_route", "api_key")
// defer cli.Close()

// device_id is the identity for the device
// payload can be an empty string if served as a heartbeat
resp, err := cli.PublishEventSync(
    &Header{
        DeviceID: "device_id",
    },
    []byte("payload"),
)
```

### Publish Event Asynchronously

``` go
import (
	"github.com/machinefi/w3bstream-client-go/client"
)

resp, err := cli.PublishEventSync(
    &Header{
        DeviceID: "device_id",
    },
    []byte("payload"),
)
```

### Publish Event Asynchronously with Error Handler

``` go
import (
	"github.com/machinefi/w3bstream-client-go/client"
)

cli := NewClient("http_route", "api_key", WithErrHandler(func(err error) {
        fmt.Println(err)
    }))

resp, err := cli.PublishEventSync(
    &Header{
        DeviceID: "device_id",
    },
    []byte("payload"),
)
```