package main

import (
	"io/ioutil"
	"log"
	"path"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestSecureDockerFile(t *testing.T) {

	const inputDirectory = "./testfiles/dockerfiles/input"
	const outputDirectory = "./testfiles/dockerfiles/output"
	// NOTE: http mocking is not working,
	// need to investigate this issue
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// NOTE: below hack is required to capture docker api calls
	saveTr := Tr
	defer func() { Tr = saveTr }()
	Tr = httpmock.DefaultTransport

	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://gcr.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/library/python/manifests/3.7", httpmock.NewStringResponder(200, `{
	// 	"schemaVersion": 2,
	// 	"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
	// 	"config": {
	// 	   "mediaType": "application/vnd.docker.container.image.v1+json",
	// 	   "size": 9201,
	// 	   "digest": "sha256:142b92ebe662c93bcf18cfe0c3174a76b4a7e62f290a51828aaa9eb630569008"
	// 	},
	// 	"layers": [
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 54999406,
	// 		  "digest": "sha256:d836772a1c1f9c4b1f280fb2a98ace30a4c4c87370f89aa092b35dfd9556278a"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 5156110,
	// 		  "digest": "sha256:66a9e63c657ad881997f5165c0826be395bfc064415876b9fbaae74bcb5dc721"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 10876416,
	// 		  "digest": "sha256:d1989b6e74cfdda1591b9dd23be47c5caeb002b7a151379361ec0c3f0e6d0e52"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 54579006,
	// 		  "digest": "sha256:c28818711e1ed38df107014a20127b41491b224d7aed8aa7066b55552d9600d2"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 196774352,
	// 		  "digest": "sha256:5084fa7ebd744165b15df008a9c14db7fc3d6af34cce64ba85bbaa348af594a3"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 6290793,
	// 		  "digest": "sha256:7f162c881e4f4673d8c7093e578c2dff9d249bc73fe24db52f9eb05c0575fa38"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 15481270,
	// 		  "digest": "sha256:fa5c3534431da089d8d042efd57be7f00a6e9f9b0fc536bd4b1120f146d26f15"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 234,
	// 		  "digest": "sha256:30596f5e8e78d938757214af98c9ddede83f1f8a3e10e55145f8c643d09f399d"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 2874571,
	// 		  "digest": "sha256:4861fe1f6551067ce2f2ae0b1201b3bacc356985cc986eab5c00b441c193fa10"
	// 	   }
	// 	]
	//  }`))

	tests := []struct {
		fileName  string
		isChanged bool
	}{
		{fileName: "Dockerfile-not-pinned", isChanged: true},
		{fileName: "Dockerfile-not-pinned-as", isChanged: true},
	}

	for _, test := range tests {

		input, err := ioutil.ReadFile(path.Join(inputDirectory, test.fileName))
		if err != nil {
			log.Fatal(err)
		}

		output, err := SecureDockerFile(string(input))
		if err != nil {
			t.Fatalf("Error not expected: %s", err)
		}

		expectedOutput, err := ioutil.ReadFile(path.Join(outputDirectory, test.fileName))

		if err != nil {
			log.Fatal(err)
		}

		if string(expectedOutput) != output.FinalOutput {
			t.Errorf("test failed %s did not match expected output\n%s", test.fileName, output.FinalOutput)
		}

		if output.IsChanged != test.isChanged {
			t.Errorf("test failed %s did not match IsChanged, Expected: %v Got: %v", test.fileName, test.isChanged, output.IsChanged)

		}

	}

}

func Test_getSHA(t *testing.T) {

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	saveTr := Tr
	defer func() { Tr = saveTr }()
	Tr = httpmock.DefaultTransport

	httpmock.RegisterResponder("GET", "https://ghcr.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://gcr.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/",
		httpmock.NewStringResponder(200, `{
	}`))

	httpmock.RegisterResponder("GET", "https://index.docker.io/v2/library/python/manifests/3.7", httpmock.NewStringResponder(200, `{
	// 	"schemaVersion": 2,
	// 	"mediaType": "application/vnd.docker.distribution.manifest.v2+json",
	// 	"config": {
	// 	   "mediaType": "application/vnd.docker.container.image.v1+json",
	// 	   "size": 9201,
	// 	   "digest": "sha256:142b92ebe662c93bcf18cfe0c3174a76b4a7e62f290a51828aaa9eb630569008"
	// 	},
	// 	"layers": [
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 54999406,
	// 		  "digest": "sha256:d836772a1c1f9c4b1f280fb2a98ace30a4c4c87370f89aa092b35dfd9556278a"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 5156110,
	// 		  "digest": "sha256:66a9e63c657ad881997f5165c0826be395bfc064415876b9fbaae74bcb5dc721"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 10876416,
	// 		  "digest": "sha256:d1989b6e74cfdda1591b9dd23be47c5caeb002b7a151379361ec0c3f0e6d0e52"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 54579006,
	// 		  "digest": "sha256:c28818711e1ed38df107014a20127b41491b224d7aed8aa7066b55552d9600d2"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 196774352,
	// 		  "digest": "sha256:5084fa7ebd744165b15df008a9c14db7fc3d6af34cce64ba85bbaa348af594a3"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 6290793,
	// 		  "digest": "sha256:7f162c881e4f4673d8c7093e578c2dff9d249bc73fe24db52f9eb05c0575fa38"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 15481270,
	// 		  "digest": "sha256:fa5c3534431da089d8d042efd57be7f00a6e9f9b0fc536bd4b1120f146d26f15"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 234,
	// 		  "digest": "sha256:30596f5e8e78d938757214af98c9ddede83f1f8a3e10e55145f8c643d09f399d"
	// 	   },
	// 	   {
	// 		  "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
	// 		  "size": 2874571,
	// 		  "digest": "sha256:4861fe1f6551067ce2f2ae0b1201b3bacc356985cc986eab5c00b441c193fa10"
	// 	   }
	// 	]
	//  }`))

	type args struct {
		image string
		tag   string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{name: "test1", args: args{image: "python", tag: "3.7"}, want: "sha256:500cc991c42e14eda7d4cc56508ac4331f829a4497724ed5f5f964a9e1e33367", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getSHA(tt.args.image, tt.args.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("getSHA() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getSHA() = %v, want %v", got, tt.want)
			}
		})
	}
}
