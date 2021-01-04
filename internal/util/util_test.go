package util

import "testing"

func TestGetStandardImageTag(t *testing.T) {
	assertCorrectMessage := func(t testing.TB, got, want string) {
		t.Helper()
		if got != want {
			t.Errorf("got %q want %q", got, want)
		}
	}

	imageTagTests := []struct {
		description string
		imageName   string
		imageTag    string
		expected    string
	}{
		{
			"image: crunchy-postgres-ha, tag: centos8-12.4-4.5.0",
			"crunchy-postgres-ha",
			"centos8-12.4-4.5.0",
			"centos8-12.4-4.5.0",
		}, {
			"image: crunchy-postgres-gis-ha, tag: centos8-12.4-3.0-4.5.0",
			"crunchy-postgres-gis-ha",
			"centos8-12.4-3.0-4.5.0",
			"centos8-12.4-4.5.0",
		}, {
			"image: crunchy-postgres-ha, tag: centos8-12.4-4.5.0-beta.1",
			"crunchy-postgres-ha",
			"centos8-12.4-4.5.0-beta.1",
			"centos8-12.4-4.5.0-beta.1",
		}, {
			"image: crunchy-postgres-gis-ha, tag: centos8-12.4-3.0-4.5.0-beta.2",
			"crunchy-postgres-gis-ha",
			"centos8-12.4-3.0-4.5.0-beta.2",
			"centos8-12.4-4.5.0-beta.2",
		}, {
			"image: crunchy-postgres-ha, tag: centos8-9.5.23-4.5.0-rc.1",
			"crunchy-postgres-ha",
			"centos8-9.5.23-4.5.0-rc.1",
			"centos8-9.5.23-4.5.0-rc.1",
		}, {
			"image: crunchy-postgres-gis-ha, tag: centos8-9.5.23-2.4-4.5.0-rc.1",
			"crunchy-postgres-gis-ha",
			"centos8-9.5.23-2.4-4.5.0-rc.1",
			"centos8-9.5.23-4.5.0-rc.1",
		}, {
			"image: crunchy-postgres-gis-ha, tag: centos8-13.0-3.0-4.5.0-rc.1",
			"crunchy-postgres-gis-ha",
			"centos8-13.0-3.0-4.5.0-rc.1",
			"centos8-13.0-4.5.0-rc.1",
		}, {
			"image: crunchy-postgres-gis-ha, tag: centos8-custom123",
			"crunchy-postgres-gis-ha",
			"centos8-custom123",
			"centos8-custom123",
		}, {
			"image: crunchy-postgres-gis-ha, tag: centos8-custom123-moreinfo-789",
			"crunchy-postgres-gis-ha",
			"centos8-custom123-moreinfo-789",
			"centos8-custom123-moreinfo-789",
		},
	}

	for _, itt := range imageTagTests {
		t.Run(itt.description, func(t *testing.T) {
			got := GetStandardImageTag(itt.imageName, itt.imageTag)
			want := itt.expected
			assertCorrectMessage(t, got, want)
		})
	}
}
