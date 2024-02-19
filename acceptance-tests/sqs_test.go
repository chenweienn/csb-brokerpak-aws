package acceptance_tests_test

import (
	"csbbrokerpakaws/acceptance-tests/helpers/apps"
	"csbbrokerpakaws/acceptance-tests/helpers/random"
	"csbbrokerpakaws/acceptance-tests/helpers/services"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SQS", Label("sqs"), func() {
	It("can be accessed by an app", func() {
		By("creating a service instance")
		serviceInstance := services.CreateInstance("csb-aws-sqs", services.WithPlan("standard"))
		defer serviceInstance.Delete()

		By("pushing the unstarted app twice")
		appOne := apps.Push(apps.WithApp(apps.SQS))
		appTwo := apps.Push(apps.WithApp(apps.SQS))
		defer apps.Delete(appOne, appTwo)

		By("binding the apps to the service instance")
		bindingOneName := random.Name(random.WithPrefix("producer"))
		binding := serviceInstance.Bind(appOne, services.WithBindingName(bindingOneName))
		bindingTwoName := random.Name(random.WithPrefix("consumer"))
		serviceInstance.Bind(appTwo, services.WithBindingName(bindingTwoName))

		By("starting the apps")
		apps.Start(appOne, appTwo)

		By("checking that the app environment has a credhub reference for credentials")
		Expect(binding.Credential()).To(HaveKey("credhub-ref"))

		By("sending a message using the first app")
		message := random.Hexadecimal()
		appOne.POST(message, "/send/%s", bindingOneName)

		By("receiving the message using the second app")
		got := appTwo.GET("/receive/%s", bindingTwoName).String()
		Expect(got).To(Equal(message))
	})
})
