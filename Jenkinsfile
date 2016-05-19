#!/usr/bin/groovy
node{

  git 'https://github.com/fabric8io/exposer.git'

  kubernetes.pod('buildpod').withImage('fabric8/go-builder')
  .withEnvVar('GOPATH','/home/jenkins/workspace/workspace/go')
  .withPrivileged(true).inside {

    stage 'build binary'

    sh "mkdir -p ../go/src/github.com/fabric8io/exposer; cp -R ../${env.JOB_NAME}/. ../go/src/github.com/fabric8io/exposer/; cd ../go/src/github.com/fabric8io/exposer; make build test lint"

    sh "cp -R ../go/src/github.com/fabric8io/exposer/bin ."

    def imageName = 'exposer'
    def tag = 'latest'

    stage 'build image'
    kubernetes.image().withName(imageName).build().fromPath(".")

    stage 'tag'
    kubernetes.image().withName(imageName).tag().inRepository('docker.io/fabric8/'+imageName).force().withTag(tag)

    stage 'push'
    kubernetes.image().withName('docker.io/fabric8/'+imageName).push().withTag(tag).toRegistry()

  }
}
