`Create CRD`
`$ kubectl apply -f crd.yaml`

`now run code`
`$ go run main.go`

`Necessary Command`
`$ kubectl get crds #get the crds
$ kubectl get songs #get the cr(song)
$ kubectl get songs.music.frsarker.dev my-custom-song -n default -o yaml #Detailed information of existing cr
`
