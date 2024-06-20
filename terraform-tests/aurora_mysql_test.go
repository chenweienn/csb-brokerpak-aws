package terraformtests

import (
	"path"

	. "terraformtests/helpers"

	tfjson "github.com/hashicorp/terraform-json"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
)

var _ = Describe("Aurora mysql", Label("aurora-mysql-terraform"), Ordered, func() {
	var (
		plan                  tfjson.Plan
		terraformProvisionDir string
		defaultVars           map[string]any
	)

	BeforeEach(func() {
		defaultVars = map[string]any{
			"instance_name":                          "csb-auroramysql-test",
			"db_name":                                "csbdb",
			"labels":                                 map[string]any{"key1": "some-mysql-value"},
			"region":                                 awsRegion,
			"aws_access_key_id":                      awsAccessKeyID,
			"aws_secret_access_key":                  awsSecretAccessKey,
			"aws_vpc_id":                             awsVPCID,
			"cluster_instances":                      3,
			"serverless_min_capacity":                nil,
			"serverless_max_capacity":                nil,
			"engine_version":                         "8.0",
			"rds_subnet_group":                       "",
			"rds_vpc_security_group_ids":             "",
			"allow_major_version_upgrade":            true,
			"auto_minor_version_upgrade":             true,
			"backup_retention_period":                1,
			"preferred_backup_window":                "23:26-23:56",
			"copy_tags_to_snapshot":                  true,
			"deletion_protection":                    false,
			"db_cluster_parameter_group_name":        "",
			"enable_audit_logging":                   false,
			"cloudwatch_log_group_retention_in_days": 14,
			"cloudwatch_log_group_kms_key_id":        "",
			"monitoring_interval":                    0,
			"monitoring_role_arn":                    "",
			"performance_insights_enabled":           false,
			"performance_insights_kms_key_id":        "",
			"performance_insights_retention_period":  7,
			"instance_class":                         "db.r5.large",
			"storage_encrypted":                      true,
			"kms_key_id":                             "",
			"preferred_maintenance_end_hour":         nil,
			"preferred_maintenance_start_hour":       nil,
			"preferred_maintenance_end_min":          nil,
			"preferred_maintenance_start_min":        nil,
			"preferred_maintenance_day":              nil,
		}
	})

	BeforeAll(func() {
		terraformProvisionDir = path.Join(workingDir, "aurora-mysql/provision")
		Init(terraformProvisionDir)
	})

	Context("with Default values", func() {
		BeforeAll(func() {
			plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{}))
		})

		It("should create the right resources", func() {
			Expect(plan.ResourceChanges).To(HaveLen(9))

			Expect(ResourceChangesTypes(plan)).To(ConsistOf(
				"aws_rds_cluster_instance",
				"aws_rds_cluster_instance",
				"aws_rds_cluster_instance",
				"aws_rds_cluster",
				"random_password",
				"random_string",
				"aws_security_group_rule",
				"aws_db_subnet_group",
				"aws_security_group",
			))
		})

		It("should create a cluster_instance with the right values", func() {
			Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(MatchKeys(IgnoreExtras, Keys{
				"engine":                       Equal("aurora-mysql"),
				"identifier":                   Equal("csb-auroramysql-test-0"),
				"instance_class":               Equal("db.r5.large"),
				"db_subnet_group_name":         Equal("csb-auroramysql-test-p-sn"),
				"auto_minor_version_upgrade":   BeTrue(),
				"tags":                         HaveKeyWithValue("key1", "some-mysql-value"),
				"monitoring_interval":          BeNumerically("==", 0),
				"performance_insights_enabled": BeFalse(),
				"apply_immediately":            BeTrue(),
			}))
		})

		It("should create a cluster with the right values", func() {
			Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(MatchKeys(IgnoreExtras, Keys{
				"cluster_identifier":                 Equal("csb-auroramysql-test"),
				"engine":                             Equal("aurora-mysql"),
				"database_name":                      Equal("csbdb"),
				"port":                               Equal(float64(3306)),
				"db_subnet_group_name":               Equal("csb-auroramysql-test-p-sn"),
				"skip_final_snapshot":                BeTrue(),
				"serverlessv2_scaling_configuration": BeEmpty(),
				"allow_major_version_upgrade":        BeTrue(),
				"tags":                               HaveKeyWithValue("key1", "some-mysql-value"),
				"backup_retention_period":            BeNumerically("==", 1),
				"preferred_backup_window":            Equal("23:26-23:56"),
				"copy_tags_to_snapshot":              BeTrue(),
				"deletion_protection":                BeFalse(),
				"enabled_cloudwatch_logs_exports":    BeNil(),
				"storage_encrypted":                  BeTrue(),
				"apply_immediately":                  BeTrue(),
			}))
		})
	})

	When("cluster_instances is 0", func() {
		BeforeAll(func() {
			plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
				"cluster_instances": 0,
			}))
		})

		It("should not create any cluster_instance", func() {
			Expect(plan.ResourceChanges).To(HaveLen(6))

			Expect(ResourceChangesTypes(plan)).To(ConsistOf(
				"aws_rds_cluster",
				"random_password",
				"random_string",
				"aws_security_group_rule",
				"aws_db_subnet_group",
				"aws_security_group",
			))
		})

		It("should create a cluster with the right values", func() {
			Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(MatchKeys(IgnoreExtras, Keys{
				"cluster_identifier":   Equal("csb-auroramysql-test"),
				"engine":               Equal("aurora-mysql"),
				"database_name":        Equal("csbdb"),
				"port":                 Equal(float64(3306)),
				"db_subnet_group_name": Equal("csb-auroramysql-test-p-sn"),
				"skip_final_snapshot":  BeTrue(),
			}))
		})
	})

	When("rds_vpc_security_group_ids is passed", func() {
		BeforeAll(func() {
			plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
				"rds_vpc_security_group_ids": "group1,group2,group3",
			}))
		})

		It("should use the ids passed and not create new security groups", func() {
			Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(
				MatchKeys(IgnoreExtras, Keys{
					"vpc_security_group_ids": ConsistOf("group1", "group2", "group3"),
				}))
			Expect(ResourceCreationForType(plan, "aws_security_group")).To(BeEmpty())
			Expect(ResourceCreationForType(plan, "aws_security_group_rule")).To(BeEmpty())
		})
	})

	When("rds_subnet_group is passed", func() {
		BeforeAll(func() {
			plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
				"rds_subnet_group": "some-other-group",
			}))
		})

		It("should use the ids passed and not create new db subnet group", func() {
			Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(
				MatchKeys(IgnoreExtras, Keys{
					"db_subnet_group_name": Equal("some-other-group"),
				}))
			Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(
				MatchKeys(IgnoreExtras, Keys{
					"db_subnet_group_name": Equal("some-other-group"),
				}))

			Expect(ResourceCreationForType(plan, "rds_subnet_group")).To(BeEmpty())
		})
	})

	When("enable_audit_logging is enabled", func() {
		BeforeAll(func() {
			plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
				"db_cluster_parameter_group_name": "db-cluster-parameter-group",
				"enable_audit_logging":            true,
			}))
		})

		It("should enable audit logging", func() {
			Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(
				MatchKeys(IgnoreExtras, Keys{
					"db_cluster_parameter_group_name": Equal("db-cluster-parameter-group"),
					"enabled_cloudwatch_logs_exports": ConsistOf("audit"),
				}))
		})

		It("should create a aws_cloudwatch_log_group", func() {
			Expect(plan.ResourceChanges).To(HaveLen(10))

			Expect(ResourceChangesTypes(plan)).To(ContainElement("aws_cloudwatch_log_group"))
		})
	})

	When("performance insights is enabled", func() {
		BeforeAll(func() {
			plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
				"performance_insights_enabled":          true,
				"performance_insights_kms_key_id":       "arn:aws:kms:" + awsRegion + ":649758297924:key/ebbb4ecc-ddfb-4e2f-8e93-c96d7bc43daa",
				"performance_insights_retention_period": 7,
			}))
		})

		It("should use the ids passed and not create new db subnet group", func() {
			Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(
				MatchKeys(IgnoreExtras, Keys{
					"performance_insights_enabled":          BeTrue(),
					"performance_insights_kms_key_id":       Equal("arn:aws:kms:" + awsRegion + ":649758297924:key/ebbb4ecc-ddfb-4e2f-8e93-c96d7bc43daa"),
					"performance_insights_retention_period": BeNumerically("==", 7),
				}),
			)
		})
	})

	When("custom key is specified", func() {
		BeforeAll(func() {
			plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
				"kms_key_id": "arn:aws:kms:us-west-9:123456789012:key/900dd091-2b79-47d2-aee8-c92e17cc7cce",
			}))
		})

		It("should use the key on the cluster", func() {
			Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(
				MatchKeys(IgnoreExtras, Keys{
					"kms_key_id": Equal("arn:aws:kms:us-west-9:123456789012:key/900dd091-2b79-47d2-aee8-c92e17cc7cce"),
				}))
		})

	})

	Context("serverless", func() {
		BeforeAll(func() {
			plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
				"serverless_min_capacity": 0.5,
				"serverless_max_capacity": 11.0,
				"instance_class":          "db.serverless",
			}))
		})

		It("passes the min_capacity, max_capacity and correct instance_class", func() {
			Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(MatchKeys(IgnoreExtras, Keys{
				"serverlessv2_scaling_configuration": ConsistOf(MatchAllKeys(Keys{
					"min_capacity": Equal(0.5),
					"max_capacity": Equal(11.0),
				})),
			}))

			Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(MatchKeys(IgnoreExtras, Keys{
				"instance_class": Equal("db.serverless"),
			}))
		})
	})

	Context("preferred_maintenance_window", func() {
		When("no window is set", func() {
			BeforeAll(func() {
				plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{}))
			})
			It("should not be passed", func() {
				Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(Not(HaveKey("preferred_maintenance_window")))
				Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(Not(HaveKey("preferred_maintenance_window")))
			})
		})

		When("preferred maintenance window specified with all values", func() {
			BeforeAll(func() {
				plan = ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
					"preferred_maintenance_day":        "Mon",
					"preferred_maintenance_start_hour": "01",
					"preferred_maintenance_end_hour":   "02",
					"preferred_maintenance_start_min":  "03",
					"preferred_maintenance_end_min":    "04",
				}))
			})

			It("should pass the correct window", func() {
				Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(
					MatchKeys(IgnoreExtras, Keys{
						"preferred_maintenance_window": Equal("mon:01:03-mon:02:04"),
					}),
				)

				Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(
					MatchKeys(IgnoreExtras, Keys{
						"preferred_maintenance_window": Equal("mon:01:03-mon:02:04"),
					}),
				)
			})

		})
	})

	Context("auto_minor_version_upgrade", func() {
		When("is enabled and a not major version is selected", func() {
			It("should complain about postcondition", func() {
				session, _ := FailPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
					"auto_minor_version_upgrade": true,
					"engine_version":             "8.0.mysql_aurora.3.04.2",
				}))

				Expect(session.ExitCode()).NotTo(Equal(0))
				msgs := string(session.Out.Contents())
				Expect(msgs).To(ContainSubstring(`Error: Resource postcondition failed`))
				Expect(msgs).To(ContainSubstring(`A Major engine version should be specified when auto_minor_version_upgrade is enabled. Expected engine version: 8.0 - got: 8.0.mysql_aurora.3.04.2`))

				session, _ = FailPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
					"auto_minor_version_upgrade": true,
					"engine_version":             "5.7.mysql_aurora.2.07.10",
				}))

				Expect(session.ExitCode()).NotTo(Equal(0))
				msgs = string(session.Out.Contents())
				Expect(msgs).To(ContainSubstring(`Error: Resource postcondition failed`))
				Expect(msgs).To(ContainSubstring(`A Major engine version should be specified when auto_minor_version_upgrade is enabled. Expected engine version: 5.7 - got: 5.7.mysql_aurora.2.07.10`))
			})
		})

		When("is disabled and a major version is selected", func() {
			It("should not complain about postcondition", func() {
				plan := ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
					"auto_minor_version_upgrade": false,
					"engine_version":             "5.7",
				}))

				Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(
					MatchKeys(IgnoreExtras, Keys{
						"engine_version": Equal("5.7"),
					}),
				)

				Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(
					MatchKeys(IgnoreExtras, Keys{
						"auto_minor_version_upgrade": BeFalse(),
						"engine_version":             Equal("5.7"),
					}),
				)
			})
		})

		When("is disabled and a minor version is selected", func() {
			It("should not complain about postcondition and create the instance", func() {
				plan := ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
					"auto_minor_version_upgrade": false,
					"engine_version":             "5.7.mysql_aurora.2.07.10",
				}))

				Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(
					MatchKeys(IgnoreExtras, Keys{
						"engine_version": Equal("5.7.mysql_aurora.2.07.10"),
					}),
				)

				Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(
					MatchKeys(IgnoreExtras, Keys{
						"auto_minor_version_upgrade": BeFalse(),
						"engine_version":             Equal("5.7.mysql_aurora.2.07.10"),
					}),
				)
			})
		})

		When("is enabled", func() {
			It("should not complain about postcondition and create the instance if engine version is null", func() {
				plan := ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
					"auto_minor_version_upgrade": true,
					"engine_version":             nil,
				}))

				Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(
					MatchKeys(IgnoreExtras, Keys{
						"engine": Equal("aurora-mysql"),
					}),
				)

				Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(
					MatchKeys(IgnoreExtras, Keys{
						"auto_minor_version_upgrade": BeTrue(),
					}),
				)
			})

			It("should not complain about postcondition and create the instance if engine version is empty", func() {
				plan := ShowPlan(terraformProvisionDir, buildVars(defaultVars, map[string]any{
					"auto_minor_version_upgrade": true,
					"engine_version":             "",
				}))

				Expect(AfterValuesForType(plan, "aws_rds_cluster")).To(
					MatchKeys(IgnoreExtras, Keys{
						"engine": Equal("aurora-mysql"),
					}),
				)

				Expect(AfterValuesForType(plan, "aws_rds_cluster_instance")).To(
					MatchKeys(IgnoreExtras, Keys{
						"auto_minor_version_upgrade": BeTrue(),
					}),
				)
			})
		})
	})
})
