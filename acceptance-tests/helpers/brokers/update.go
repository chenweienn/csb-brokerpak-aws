package brokers

import (
	"fmt"
	"slices"

	"csbbrokerpakaws/acceptance-tests/helpers/apps"
	"csbbrokerpakaws/acceptance-tests/helpers/cf"
)

func (b *Broker) Hostname() string {
	return fmt.Sprintf("http://%s.cloud-service-broker.service.internal:8080", b.Name)
}

func (b *Broker) UpdateBrokerToVmBroker() {
	cf.Run("update-service-broker", b.Name, b.username, b.password, b.Hostname())
}

func (b *Broker) UpdateBroker(dir string, env ...apps.EnvVar) {
	b.envExtras = slices.Concat(b.envExtras, b.latestEnv(), env)

	b.app.Push(
		apps.WithName(b.Name),
		apps.WithDir(dir),
		apps.WithStartedState(),
		apps.WithManifest(newManifest(
			withName(b.Name),
			withEnv(b.env()...),
		)),
	)

	cf.Run("update-service-broker", b.Name, b.username, b.password, b.app.URL)
}

func (b *Broker) UpdateEnv(env ...apps.EnvVar) {
	WithEnv(env...)(b)
	b.app.SetEnv(b.env()...)
	b.app.Restart()

	cf.Run("update-service-broker", b.Name, b.username, b.password, b.app.URL)
}

func (b *Broker) UpdateEncryptionSecrets(secrets ...EncryptionSecret) {
	WithEncryptionSecrets(secrets...)
	b.app.SetEnv(b.env()...)

	cf.Run("update-service-broker", b.Name, b.username, b.password, b.app.URL)
}
