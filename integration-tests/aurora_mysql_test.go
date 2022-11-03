package integration_test

import (
	"fmt"

	testframework "github.com/cloudfoundry/cloud-service-broker/brokerpaktestframework"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var customAuroraMySQLPlans = []map[string]any{
	customAuroraMySQLPlan,
}

var customAuroraMySQLPlan = map[string]any{
	"name":        "custom-sample",
	"id":          "10b2bd92-2a0b-11ed-b70f-c7c5cf3bb719",
	"description": "Default Aurora MySQL plan",
	"metadata": map[string]any{
		"displayName": "custom-sample",
	},
}

var _ = Describe("Aurora MySQL", Label("aurora-mysql"), func() {
	const serviceName = "csb-aws-aurora-mysql"

	BeforeEach(func() {
		Expect(mockTerraform.SetTFState([]testframework.TFStateValue{})).To(Succeed())
	})

	AfterEach(func() {
		Expect(mockTerraform.Reset()).To(Succeed())
	})

	It("should publish in the catalog", func() {
		catalog, err := broker.Catalog()
		Expect(err).NotTo(HaveOccurred())

		service := testframework.FindService(catalog, serviceName)
		Expect(service.ID).NotTo(BeNil())
		Expect(service.Name).NotTo(BeNil())
		Expect(service.Tags).To(ConsistOf("aws", "mysql", "aurora", "beta"))
		Expect(service.Metadata.ImageUrl).NotTo(BeNil())
		Expect(service.Metadata.DisplayName).NotTo(BeNil())
		Expect(service.Plans).To(
			ConsistOf(
				MatchFields(IgnoreExtras, Fields{"Name": Equal("custom-sample")}),
			),
		)
	})

	Describe("provisioning", func() {
		DescribeTable("should check property constraints",
			func(params map[string]any, expectedErrorMsg string) {
				_, err := broker.Provision(serviceName, "custom-sample", params)

				Expect(err).To(MatchError(ContainSubstring(expectedErrorMsg)))
			},
			Entry(
				"invalid region",
				map[string]any{"region": "-Asia-northeast1"},
				"region: Does not match pattern '^[a-z][a-z0-9-]+$'",
			),
			Entry(
				"instance name minimum length is 6 characters",
				map[string]any{"instance_name": stringOfLen(5)},
				"instance_name: String length must be greater than or equal to 6",
			),
			Entry(
				"instance name maximum length is 98 characters",
				map[string]any{"instance_name": stringOfLen(99)},
				"instance_name: String length must be less than or equal to 98",
			),
			Entry(
				"instance name invalid characters",
				map[string]any{"instance_name": ".aaaaa"},
				"instance_name: Does not match pattern '^[a-z][a-z0-9-]+$'",
			),
			Entry(
				"database name maximum length is 64 characters",
				map[string]any{"db_name": stringOfLen(65)},
				"db_name: String length must be less than or equal to 64",
			),
		)

		It("should provision a plan", func() {
			instanceID, err := broker.Provision(serviceName, "custom-sample", nil)
			Expect(err).NotTo(HaveOccurred())

			Expect(mockTerraform.FirstTerraformInvocationVars()).To(SatisfyAll(
				HaveKeyWithValue("instance_name", fmt.Sprintf("csb-auroramysql-%s", instanceID)),
				HaveKeyWithValue("cluster_instances", BeNumerically("==", 3)),
				HaveKeyWithValue("db_name", "csbdb"),
				HaveKeyWithValue("region", "us-west-2"),
				HaveKeyWithValue("allow_major_version_upgrade", BeTrue()),
				HaveKeyWithValue("auto_minor_version_upgrade", BeTrue()),
				HaveKeyWithValue("rds_vpc_security_group_ids", BeEmpty()),
				HaveKeyWithValue("rds_subnet_group", BeEmpty()),
				HaveKeyWithValue("labels", HaveKeyWithValue("pcf-instance-id", instanceID)),
				HaveKeyWithValue("deletion_protection", BeFalse()),
			))
		})

		It("should allow properties to be set on provision", func() {
			_, err := broker.Provision(serviceName, "custom-sample", map[string]any{
				"instance_name":               "csb-aurora-mysql-fake-name",
				"cluster_instances":           12,
				"region":                      "africa-north-4",
				"db_name":                     "fake-db-name",
				"serverless_min_capacity":     0.2,
				"serverless_max_capacity":     100,
				"engine_version":              "8.0.mysql_aurora.3.02.0",
				"allow_major_version_upgrade": false,
				"auto_minor_version_upgrade":  false,
				"rds_vpc_security_group_ids":  "group1,group2",
				"rds_subnet_group":            "some-other-subnet",
				"deletion_protection":         true,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(mockTerraform.FirstTerraformInvocationVars()).To(
				SatisfyAll(
					HaveKeyWithValue("instance_name", "csb-aurora-mysql-fake-name"),
					HaveKeyWithValue("cluster_instances", BeNumerically("==", 12)),
					HaveKeyWithValue("db_name", "fake-db-name"),
					HaveKeyWithValue("region", "africa-north-4"),
					HaveKeyWithValue("serverless_min_capacity", BeNumerically("==", 0.2)),
					HaveKeyWithValue("serverless_max_capacity", BeNumerically("==", 100)),
					HaveKeyWithValue("engine_version", "8.0.mysql_aurora.3.02.0"),
					HaveKeyWithValue("allow_major_version_upgrade", false),
					HaveKeyWithValue("auto_minor_version_upgrade", false),
					HaveKeyWithValue("rds_vpc_security_group_ids", "group1,group2"),
					HaveKeyWithValue("rds_subnet_group", "some-other-subnet"),
					HaveKeyWithValue("deletion_protection", true),
				),
			)
		})
	})

	Describe("updating instance", func() {
		var instanceID string

		BeforeEach(func() {
			var err error
			instanceID, err = broker.Provision(serviceName, "custom-sample", nil)

			Expect(err).NotTo(HaveOccurred())
		})

		DescribeTable(
			"preventing updates with `prohibit_update` as it can force resource replacement or re-creation",
			func(prop string, value any) {
				err := broker.Update(instanceID, serviceName, "custom-sample", map[string]any{prop: value})

				Expect(err).To(MatchError(
					ContainSubstring(
						"attempt to update parameter that may result in service instance re-creation and data loss",
					),
				))

				const initialProvisionInvocation = 1
				Expect(mockTerraform.ApplyInvocations()).To(HaveLen(initialProvisionInvocation))
			},
			Entry("region", "region", "no-matter-what-region"),
			Entry("instance_name", "instance_name", "marmaduke"),
			Entry("db_name", "db_name", "some-new-name"),
			Entry("rds_subnet_group", "rds_subnet_group", "some-new-subnet-name"),
		)

		DescribeTable(
			"allowed updates",
			func(prop string, value any) {
				Expect(broker.Update(instanceID, serviceName, "custom-sample", map[string]any{prop: value})).To(Succeed())
			},
			Entry("cluster_instances", "cluster_instances", 11),
			Entry("serverless_min_capacity", "serverless_min_capacity", 1),
			Entry("serverless_max_capacity", "serverless_max_capacity", 30),
			Entry("engine_version", "engine_version", "8.0.mysql_aurora.3.02.0"),
			Entry("allow_major_version_upgrade", "allow_major_version_upgrade", false),
			Entry("auto_minor_version_upgrade", "auto_minor_version_upgrade", false),
			Entry("rds_vpc_security_group_ids", "rds_vpc_security_group_ids", "group3"),
		)
	})
})
