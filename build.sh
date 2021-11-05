protoc -I . --go_out . ./message.proto
mockery --dir ./base --name ISession --structname MockSession --output ./base --outpkg base --filename mock_session.go
