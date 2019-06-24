To run the nsq_tail utility to see the published messages:

go test -run TestEventRead  -nsqd-tcp-address="127.0.0.1:14150" -topic=alltopic -print-topic=true
