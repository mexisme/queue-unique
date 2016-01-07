package queue

import (
	//"os"
	"time"

	log "gopkg.in/Sirupsen/logrus.v0"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type Fake struct {
	id    string
	value string
}

var _ = Describe("QueueUnique", func() {
	//log.SetOutput(os.Stderr)
	log.SetOutput(GinkgoWriter)
	log.SetLevel(log.DebugLevel)

	var uq *UniqueQueue
	matcherID := func(fake interface{}) string {
		return fake.(*Fake).id
	}
	faked := &Fake{id: "1234", value: "5678"}

	AfterEach(func() {
		uq.Close()
		uq = nil
	})

	It("should initialise to empty", func() {
		uq = (&UniqueQueue{
			MatcherID: matcherID,
		}).Init()

		Expect(uq.uniqueIDs).To(BeEmpty())
		Expect(uq.feeder).To(BeEmpty())
		Expect(uq.In).To(BeEmpty())
		Expect(uq.Out).To(BeEmpty())
	})

	var _ = Context("when pushing to the feeder queue", func() {
		It("should panic when MatcherID has not been provided", func() {
			uq = (&UniqueQueue{}).Init()
			Expect(func() {
				uq.pushUnique(faked)
			}).To(Panic())
		})

		var _ = Context("", func() {
			BeforeEach(func() {
				uq = (&UniqueQueue{
					MatcherID: matcherID,
				}).Init()
			})

			It("should push a request successfully", func(done Done) {
				Expect(uq.pushUnique(faked)).To(BeTrue())

				close(done)
			}, 0.2)

			var _ = Context("", func() {
				BeforeEach(func() {
					uq.pushUnique(faked)
				})

				It("should store the ID in uniqueUrls", func(done Done) {
					Expect(uq.uniqueIDs[faked.id]).To(BeTrue())

					close(done)
				}, 0.2)

				It("should store the ID in the feeder queue", func(done Done) {
					Expect(<-uq.feeder).To(Equal(faked))

					close(done)
				}, 0.2)

				It("should have an empty queue at the end", func(done Done) {
					Expect(<-uq.feeder).To(Equal(faked))
					Expect(uq.feeder).To(BeEmpty())

					close(done)
				}, 0.2)
			})
		})
	})

	var _ = Context("when popping from the feeder queue", func() {
		var outFake *Fake

		BeforeEach(func() {
			uq = (&UniqueQueue{
				MatcherID: matcherID,
			}).Init()

			uq.pushUnique(faked)

			uq.popTo(func(fake interface{}) {
				outFake = fake.(*Fake)
			})
		})

		AfterEach(func() {
			outFake = nil
		})

		It("should pop the same value as pushed", func(done Done) {
			Expect(outFake).To(Equal(faked))

			close(done)
		}, 0.2)

		It("should have an empty feeder queue when popped", func() {
			Expect(uq.feeder).To(BeEmpty())
		})

		It("should have an empty uniqueIDs when popped", func() {
			Expect(uq.uniqueIDs).To(BeEmpty())
		})
	})

	var _ = Context("when Run()", func() {
		fakedArr := [](*Fake){
			&Fake{id: "1", value: "0123"},
			&Fake{id: "2", value: "4567"},
			&Fake{id: "3", value: "89ab"},
			&Fake{id: "4", value: "cdef"},
		}
		var inQ, outQ chan interface{}

		AfterEach(func() {
			// It seems it's a bit time-sensitive:
			time.Sleep(1 * time.Millisecond)

			close(inQ)
			close(outQ)
		})

		It("should panic when the Out queue < 1", func() {
			inQ, outQ = make(chan interface{}, 100), make(chan interface{})
			uq = (&UniqueQueue{
				MatcherID: matcherID,
				In:        inQ,
				Out:       outQ,
			}).Init()

			Expect(func() {
				uq.Run()
			}).To(Panic())
		})

		It("should forward items from In to Out channels", func(done Done) {
			inQ, outQ = make(chan interface{}, 100), make(chan interface{}, 100)
			uq = (&UniqueQueue{
				MatcherID: matcherID,
				In:        inQ,
				Out:       outQ,
			}).Init()

			for _, f := range fakedArr {
				inQ <- f
			}
			Eventually(len(uq.In)).Should(Equal(len(fakedArr)))
			Eventually(len(uq.Out)).Should(Equal(0))
			Eventually(len(inQ)).Should(Equal(len(fakedArr)))
			Eventually(len(outQ)).Should(Equal(0))

			uq.Run()

			// There's a fractional "wait" we have to do to let the queue feed through:
			time.Sleep(1 * time.Millisecond)

			Eventually(len(uq.In)).Should(Equal(0))
			Eventually(len(uq.Out)).Should(Equal(len(fakedArr)))
			Eventually(len(inQ)).Should(Equal(0))
			Eventually(len(outQ)).Should(Equal(len(fakedArr)))

			for _, f := range fakedArr {
				Eventually(outQ).Should(Receive(Equal(f)))
			}

			Eventually(len(uq.In)).Should(Equal(0))
			Eventually(len(uq.Out)).Should(Equal(0))
			Eventually(len(uq.feeder)).Should(Equal(0))
			Eventually(len(inQ)).Should(Equal(0))
			Eventually(len(outQ)).Should(Equal(0))

			close(done)
		}, 0.2)

		It("should dedupe repeated items from the In queue", func(done Done) {
			inQ, outQ = make(chan interface{}, 100), make(chan interface{}, 1)
			uq = (&UniqueQueue{
				MatcherID: matcherID,
				In:        inQ,
				Out:       outQ,
			}).Init()
			dupes := 5
			fakedArrLen := len(fakedArr)
			fakedArrDupesOutQLen := (fakedArrLen * dupes) + cap(outQ)

			// Because the outQ has a buffer-size, we have to fill it before we start the test
			for i := 0; i < cap(outQ); i++ {
				inQ <- &Fake{id: "filler", value: "filler"}
			}

			for i := 0; i < dupes; i++ {
				for _, f := range fakedArr {
					inQ <- f
				}
			}

			Eventually(len(uq.In)).Should(Equal(fakedArrDupesOutQLen))
			Eventually(len(uq.Out)).Should(Equal(0))
			Eventually(len(inQ)).Should(Equal(fakedArrDupesOutQLen))
			Eventually(len(outQ)).Should(Equal(0))

			uq.Run()

			// There's a fractional "wait" we have to do to let the queue feed through:
			time.Sleep(1 * time.Millisecond)

			Eventually(len(uq.uniqueIDs)).Should(Equal(fakedArrLen), "uq.uniqueIDs %#v", uq.uniqueIDs)
			Eventually(len(uq.In)).Should(Equal(0))
			Eventually(len(inQ)).Should(Equal(0))

			// Empty-out the filler entries
			for i := 0; i < cap(outQ); i++ {
				Eventually(outQ).Should(Receive(Equal(
					&Fake{id: "filler", value: "filler"},
				)))
			}

			for _, f := range fakedArr {
				Eventually(outQ).Should(Receive(Equal(f)))
			}

			Eventually(len(uq.uniqueIDs)).Should(Equal(0), "uq.uniqueIDs %#v", uq.uniqueIDs)
			Eventually(len(uq.In)).Should(Equal(0))
			Eventually(len(uq.Out)).Should(Equal(0))
			Eventually(len(uq.feeder)).Should(Equal(0))
			Eventually(len(inQ)).Should(Equal(0))
			Eventually(len(outQ)).Should(Equal(0))

			close(done)
		}, 2)
	})
})
