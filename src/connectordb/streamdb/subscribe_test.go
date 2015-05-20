package streamdb

import (
	"connectordb/streamdb/operator"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSubscribe(t *testing.T) {
	require.NoError(t, ResetTimeBatch())

	db, err := Open("postgres://127.0.0.1:52592/connectordb?sslmode=disable", "localhost:6379", "localhost:4222")
	require.NoError(t, err)
	defer db.Close()

	//Let's create a stream
	require.NoError(t, db.CreateUser("tst", "root@localhost", "mypass"))
	require.NoError(t, db.CreateDevice("tst/tst"))
	require.NoError(t, db.CreateStream("tst/tst/tst", `{"type": "string"}`))

	recvchan := make(chan operator.Message, 2)
	recvchan2 := make(chan operator.Message, 2)
	recvchan3 := make(chan operator.Message, 2)
	recvchan4 := make(chan operator.Message, 2)
	//We bind a timeout to the channel, since we want the test to fail if no messages come through
	go func() {
		time.Sleep(2 * time.Second)
		recvchan <- operator.Message{"TIMEOUT", []operator.Datapoint{}}
		recvchan2 <- operator.Message{"TIMEOUT", []operator.Datapoint{}}
		recvchan3 <- operator.Message{"TIMEOUT", []operator.Datapoint{}}
		recvchan4 <- operator.Message{"TIMEOUT", []operator.Datapoint{}}
	}()

	_, err = db.Subscribe("tst", recvchan)
	require.NoError(t, err)
	_, err = db.Subscribe("tst/tst", recvchan2)
	require.NoError(t, err)
	_, err = db.Subscribe("tst/tst/tst", recvchan3)
	require.NoError(t, err)
	_, err = db.Subscribe("tst/tst/tst/downlink", recvchan4)
	require.NoError(t, err)
	db.msg.Flush() //Just to avoid problems

	data := []operator.Datapoint{operator.Datapoint{
		Timestamp: 1.0,
		Data:      "Hello World!",
	}}

	require.NoError(t, db.InsertStream("tst/tst/tst", data))

	m := <-recvchan
	require.Equal(t, m.Stream, "tst/tst/tst")
	require.Equal(t, m.Data[0].Data, "Hello World!")
	m = <-recvchan2
	require.Equal(t, m.Stream, "tst/tst/tst")
	require.Equal(t, m.Data[0].Data, "Hello World!")
	m = <-recvchan3
	require.Equal(t, m.Stream, "tst/tst/tst")
	require.Equal(t, m.Data[0].Data, "Hello World!")

	data = []operator.Datapoint{operator.Datapoint{
		Timestamp: 2.0,
		Data:      "2",
	}}

	require.NoError(t, db.InsertStream("tst/tst/tst/downlink", data))
	db.msg.Flush()
	m = <-recvchan4
	require.Equal(t, m.Stream, "tst/tst/tst/downlink")
	require.Equal(t, m.Data[0].Data, "2")

	time.Sleep(100 * time.Millisecond)
	recvchan <- operator.Message{"GOOD", []operator.Datapoint{}}
	recvchan2 <- operator.Message{"GOOD", []operator.Datapoint{}}
	recvchan3 <- operator.Message{"GOOD", []operator.Datapoint{}}

	m = <-recvchan
	require.Equal(t, m.Stream, "GOOD", "A downlink should not be triggered")
	m = <-recvchan2
	require.Equal(t, m.Stream, "GOOD", "A downlink should not be triggered")
	m = <-recvchan3
	require.Equal(t, m.Stream, "GOOD", "A downlink should not be triggered")

}
