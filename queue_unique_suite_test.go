package queue

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestQueueUnique(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "QueueUnique Suite")
}
