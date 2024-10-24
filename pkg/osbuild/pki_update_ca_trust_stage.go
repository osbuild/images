package osbuild

func NewCAStageStage() *Stage {
	return &Stage{
		Type: "org.osbuild.pki.update-ca-trust",
	}
}
