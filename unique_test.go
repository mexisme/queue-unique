package queue

import (
	//"os"

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

	var _ = Context("", func() {
		fakedArr := [](*Fake){
			&Fake{id: "1", value: "0123"},
			&Fake{id: "2", value: "4567"},
			&Fake{id: "3", value: "89ab"},
			&Fake{id: "4", value: "cdef"},
		}
		var inQ, outQ chan interface{}

		BeforeEach(func() {
			inQ, outQ = make(chan interface{}, 100), make(chan interface{})
			uq = (&UniqueQueue{
				MatcherID: matcherID,
				In:        inQ,
				Out:       outQ,
			}).Init()
			uq.Run()
		})

		AfterEach(func() {
			close(inQ)
			close(outQ)
		})

		It("test", func(done Done) {
			for _, f := range fakedArr {
				inQ <- f
			}
			//Expect(len(uq.In)).To(Equal(len(fakedArr)))
			//Expect(len(uq.Out)).To(Equal(len(fakedArr)))
			//Expect(len(uq.feeder)).To(Equal(len(fakedArr)))
			//Expect(len(inQ)).To(Equal(len(fakedArr)))
			//Expect(len(outQ)).To(Equal(len(fakedArr)))

			for _, f := range fakedArr {
				fakeOut := (<-outQ).(*Fake)
				Expect(f).To(Equal(fakeOut))
			}

			close(done)
		}, 0.2)
	})
})
