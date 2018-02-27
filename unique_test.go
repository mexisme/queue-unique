package queue

import (
	//"os"
	"time"

	log "github.com/sirupsen/logrus"

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
	})

	It("should initialise to default buffer-size if not provided", func() {
		uq = (&UniqueQueue{
			MatcherID: matcherID,
		}).Init()

		Expect(uq.BufferSize).To(Equal(DefaultBufferSize))
	})

	It("should initialise to provided buffer-size", func() {
		uq = (&UniqueQueue{
			MatcherID:  matcherID,
			BufferSize: 2112112,
		}).Init()

		Expect(uq.BufferSize).To(Equal(2112112))
	})

	It("should use provided In queue", func() {
		inQ := make(InQueue, 1000)
		uq = (&UniqueQueue{
			MatcherID: matcherID,
			In:        inQ,
		}).Init()

		Expect(uq.In).ToNot(BeNil())
		Expect(uq.In).To(BeEmpty())
		Expect(uq.In).To(Equal(inQ))

		//close(inQ)
	})

	It("should use provided Out queue", func() {
		outQ := make(OutQueue, 1000)
		uq = (&UniqueQueue{
			MatcherID: matcherID,
			Out:       outQ,
		}).Init()

		Expect(uq.Out).ToNot(BeNil())
		Expect(uq.Out).To(BeEmpty())
		Expect(uq.Out).To(Equal(outQ))

		close(outQ)
	})

	Context("when pushing to the feeder queue", func() {
		It("should panic when MatcherID has not been provided", func() {
			uq = (&UniqueQueue{}).Init()
			Expect(func() {
				uq.pushUnique(faked)
			}).To(Panic())
		})

		Context("", func() {
			BeforeEach(func() {
				uq = (&UniqueQueue{
					MatcherID: matcherID,
				}).Init()
			})

			It("should push a request successfully", func(done Done) {
				Expect(uq.pushUnique(faked)).To(BeTrue())

				close(done)
			}, 0.2)

			Context("", func() {
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

	Context("when popping from the feeder queue", func() {
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

	Context("when Run()", func() {
		fakedArr := [](*Fake){
			&Fake{id: "1", value: "0123"},
			&Fake{id: "2", value: "4567"},
			&Fake{id: "3", value: "89ab"},
			&Fake{id: "4", value: "cdef"},
		}

		It("should panic when the In queue not made", func() {
			uq = (&UniqueQueue{
				MatcherID: matcherID,
				Out:       make(OutQueue),
			}).Init()

			Expect(func() {
				uq.Run()
			}).To(Panic())
		})

		It("should panic when the Out queue not made", func() {
			uq = (&UniqueQueue{
				MatcherID: matcherID,
				In:        make(InQueue),
			}).Init()

			Expect(func() {
				uq.Run()
			}).To(Panic())
		})

		It("should panic when the Out queue < 1", func() {
			uq = (&UniqueQueue{
				MatcherID: matcherID,
				In:        make(InQueue),
				Out:       make(OutQueue),
			}).Init()

			Expect(func() {
				uq.Run()
			}).To(Panic())
		})

		It("should forward items from In to Out channels", func(done Done) {
			uq = (&UniqueQueue{
				MatcherID: matcherID,
				In:        make(InQueue, 100),
				Out:       make(OutQueue, 100),
			}).Init()

			for _, f := range fakedArr {
				uq.In <- f
			}
			Eventually(len(uq.In)).Should(Equal(len(fakedArr)))
			Eventually(len(uq.Out)).Should(Equal(0))

			uq.Run()

			// There's a fractional "wait" we have to do to let the queue feed through:
			time.Sleep(1 * time.Millisecond)

			Eventually(len(uq.In)).Should(Equal(0))
			Eventually(len(uq.Out)).Should(Equal(len(fakedArr)))

			for _, f := range fakedArr {
				Eventually(uq.Out).Should(Receive(Equal(f)))
			}

			Eventually(len(uq.In)).Should(Equal(0))
			Eventually(len(uq.Out)).Should(Equal(0))
			Eventually(len(uq.feeder)).Should(Equal(0))

			close(done)
		}, 1)

		It("should dedupe repeated items from the In queue", func(done Done) {
			uq = (&UniqueQueue{
				MatcherID: matcherID,
				In:       make(InQueue, 100),
				Out:       make(OutQueue, 1),
			}).Init()
			dupes := 5
			fakedArrLen := len(fakedArr)
			fakedArrDupesOutQLen := (fakedArrLen * dupes) + cap(uq.Out)

			// Because the outQ has a buffer-size, we have to fill it before we start the test
			for i := 0; i < cap(uq.Out); i++ {
				uq.In <- &Fake{id: "filler", value: "filler"}
			}

			for i := 0; i < dupes; i++ {
				for _, f := range fakedArr {
					uq.In <- f
				}
			}

			Eventually(len(uq.In)).Should(Equal(fakedArrDupesOutQLen))
			Eventually(len(uq.Out)).Should(Equal(0))

			uq.Run()

			// There's a fractional "wait" we have to do to let the queue feed through:
			time.Sleep(1 * time.Millisecond)

			Eventually(len(uq.uniqueIDs)).Should(Equal(fakedArrLen), "uq.uniqueIDs %#v", uq.uniqueIDs)
			Eventually(len(uq.In)).Should(Equal(0))

			// Empty-out the filler entries
			for i := 0; i < cap(uq.Out); i++ {
				Eventually(uq.Out).Should(Receive(Equal(
					&Fake{id: "filler", value: "filler"},
				)))
			}

			for _, f := range fakedArr {
				Eventually(uq.Out).Should(Receive(Equal(f)))
			}

			Eventually(len(uq.uniqueIDs)).Should(Equal(0), "uq.uniqueIDs %#v", uq.uniqueIDs)
			Eventually(len(uq.In)).Should(Equal(0))
			Eventually(len(uq.Out)).Should(Equal(0))
			Eventually(len(uq.feeder)).Should(Equal(0))

			close(done)
		}, 2)
	})
})
