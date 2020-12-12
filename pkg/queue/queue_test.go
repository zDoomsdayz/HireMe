package queue

import (
	"fmt"
	"testing"

	. "github.com/franela/goblin"
)

func TestQueue(t *testing.T) {
	gob := Goblin(t)

	gob.Describe("Queue File Test", func() {
		history := Queue{}
		gob.It("should check for enque", func() {
			for i := 1; i <= 10; i++ {
				history.Enqueue(History{"2006-01-02 3:04PM", fmt.Sprintf("%v", i)})
				gob.Assert(history.size).Equal(i)
			}
		})

		gob.It("should check for enque cap of 10", func() {
			history.Enqueue(History{"2006-01-02 3:04PM", "11"})
			gob.Assert(history.size).Equal(10)
			history.Enqueue(History{"2006-01-02 3:04PM", "12"})
			gob.Assert(history.size).Equal(10)
			history.Enqueue(History{"2006-01-02 3:04PM", "13"})
			gob.Assert(history.size).Equal(10)
			history.Enqueue(History{"2006-01-02 3:04PM", "14"})
			gob.Assert(history.size).Equal(10)
		})

		gob.It("should check for dequeue", func() {
			for i := 9; i >= 0; i-- {
				history.Dequeue()
				gob.Assert(history.size).Equal(i)
			}
		})

		gob.It("should check for empty dequeue", func() {
			_, err := history.Dequeue()
			gob.Assert(err).Equal(fmt.Errorf("empty queue"))
			_, err = history.Dequeue()
			gob.Assert(err).Equal(fmt.Errorf("empty queue"))
		})

		gob.It("should get all history", func() {
			for i := 1; i <= 10; i++ {
				history.Enqueue(History{"2006-01-02 3:04PM", fmt.Sprintf("%v", i)})
				allHistory := history.AllHistory()
				gob.Assert(len(allHistory)).Equal(i)
				gob.Assert(allHistory[i-1]).Equal(History{"2006-01-02 3:04PM", fmt.Sprintf("%v", i)})
			}
		})
	})
}
