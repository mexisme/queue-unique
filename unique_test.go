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
			Expect(len(uq.In)).To(Equal(len(fakedArr)))
			Expect(len(uq.Out)).To(Equal(0))
			Expect(len(inQ)).To(Equal(len(fakedArr)))
			Expect(len(outQ)).To(Equal(0))

			uq.Run()

			// There's a fractional "wait" we have to do to let the queue feed through:
			time.Sleep(1 * time.Millisecond)

			Expect(len(uq.In)).To(Equal(0))
			Expect(len(uq.Out)).To(Equal(len(fakedArr)))
			Expect(len(inQ)).To(Equal(0))
			Expect(len(outQ)).To(Equal(len(fakedArr)))

			for _, f := range fakedArr {
				fakeOut := (<-outQ).(*Fake)
				Expect(f).To(Equal(fakeOut))
			}

			Expect(len(uq.In)).To(Equal(0))
			Expect(len(uq.Out)).To(Equal(0))
			Expect(len(uq.feeder)).To(Equal(0))
			Expect(len(inQ)).To(Equal(0))
			Expect(len(outQ)).To(Equal(0))

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

			for i := 0; i < dupes; i++ {
				for _, f := range fakedArr {
					inQ <- f
				}
			}

			Expect(len(uq.In)).To(Equal(len(fakedArr) * dupes))
			Expect(len(uq.Out)).To(Equal(0))
			Expect(len(inQ)).To(Equal(len(fakedArr) * dupes))
			Expect(len(outQ)).To(Equal(0))

			uq.Run()

			// There's a fractional "wait" we have to do to let the queue feed through:
			time.Sleep(1 * time.Millisecond)

			Expect(len(uq.uniqueIDs)).To(Equal(len(fakedArr)), "uq.uniqueIDs %#v", uq.uniqueIDs)
			Expect(len(uq.In)).To(Equal(0))
			Expect(len(inQ)).To(Equal(0))

			for _, f := range fakedArr {
				fakeOut := (<-outQ).(*Fake)
				Expect(f).To(Equal(fakeOut))
			}

			Expect(len(uq.In)).To(Equal(0))
			Expect(len(uq.Out)).To(Equal(0))
			Expect(len(uq.feeder)).To(Equal(0))
			Expect(len(inQ)).To(Equal(0))
			Expect(len(outQ)).To(Equal(0))

			close(done)
		}, 0.2)
	})
})
