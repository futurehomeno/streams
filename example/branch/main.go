package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"

	"github.com/futurehomeno/streams/v6"
)

func main() {
	ctx := context.Background()

	builder := streams.NewStreamBuilder()

	s := builder.Source("rand-source", newRandIntSource()).
		BranchFunc("branch", branchEvenNumberFilter, branchOddNumberFilter)

	sink1 := newCommitProcessor(1000)
	sink2 := newCommitProcessor(1000)

	// Event numbers
	s[0].Print("print-event").
		Process("commit-sink1", sink1)

	// Odd Numbers
	s[1].MapFunc("negative-mapper", negativeMapper).
		Print("print-negative").
		Process("commit-sink2", sink2)

	tp, _ := builder.Build()
	task := streams.NewTask(tp)
	task.OnError(func(err error) {
		log.Fatal(err.Error())
	})
	task.Start(ctx)
	defer task.Close()

	// Wait for SIGTERM
	waitForShutdown()
}

type randIntSource struct {
	rand *rand.Rand
}

func newRandIntSource() streams.Source {
	return &randIntSource{
		rand: rand.New(rand.NewSource(1234)),
	}
}

func (s *randIntSource) Consume() (streams.Message, error) {
	return streams.NewMessage(nil, s.rand.Intn(100)), nil
}

func (s *randIntSource) Commit(v interface{}) error {
	return nil
}

func (s *randIntSource) Close() error {
	return nil
}

type commitProcessor struct {
	pipe streams.Pipe

	batch int
	count int
}

func newCommitProcessor(batch int) streams.Processor {
	return &commitProcessor{
		batch: batch,
	}
}

func (p *commitProcessor) WithPipe(pipe streams.Pipe) {
	p.pipe = pipe
}

func (p *commitProcessor) Process(msg streams.Message) error {
	p.count++

	if p.count >= p.batch {
		return p.pipe.Commit(msg)
	}

	return p.pipe.Mark(msg)
}

func (p *commitProcessor) Commit() error {
	p.count = 0

	return nil
}

func (p *commitProcessor) Close() error {
	return nil
}

func branchOddNumberFilter(msg streams.Message) (bool, error) {
	num := msg.Value.(int)

	return num%2 == 1, nil
}

func branchEvenNumberFilter(msg streams.Message) (bool, error) {
	num := msg.Value.(int)

	return num%2 == 0, nil
}

func negativeMapper(msg streams.Message) (streams.Message, error) {
	num := msg.Value.(int)
	msg.Value = num * -1

	return msg, nil
}

// waitForShutdown blocks until a SIGINT or SIGTERM is received.
func waitForShutdown() {
	quit := make(chan os.Signal)

	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(quit)

	<-quit
}
