#!/usr/bin/groovy
node{

  git 'https://github.com/fabric8io/exposecontroller.git'

  kubernetes.pod('buildpod').withImage('fabric8/go-builder')
  .withEnvVar('GOPATH','/home/jenkins/workspace/workspace/go')
  .withPrivileged(true).inside {

    stage 'build binary'
    
    sh """
    export GOPATH=/home/jenkins/workspace/workspace/go;
    mkdir -p ../go/src/github.com/fabric8io/exposecontroller; 
    cp -R ../${env.JOB_NAME}/. ../go/src/github.com/fabric8io/exposecontroller/; 
    cd ../go/src/github.com/fabric8io/exposecontroller; make build test lint
    """

    sh "cp -R ../go/src/github.com/fabric8io/exposecontroller/bin ."

    def imageName = 'exposecontroller'
    def tag = 'latest'

    stage 'build image'
    kubernetes.image().withName(imageName).build().fromPath(".")

    stage 'tag'
    kubernetes.image().withName(imageName).tag().inRepository('docker.io/fabric8/'+imageName).force().withTag(tag)

    stage 'push'
    kubernetes.image().withName('docker.io/fabric8/'+imageName).push().withTag(tag).toRegistry()

  }
}
