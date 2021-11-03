module point-set

go 1.15

require (
	github.com/pkg/errors v0.9.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/xtaci/kcp-go/v5 v5.6.1
	google.golang.org/protobuf v1.27.1
)

replace github.com/xtaci/kcp-go/v5 v5.6.1 => ../kcp-go
