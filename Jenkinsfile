pipeline {
    agent {
        label "jenkins-go"
    }
    stages {
        stage('CI Build and Test') {
            when {
                branch 'PR-*'
            }
            steps {
                dir ('/home/jenkins/go/src/github.com/jenkins-x/exposecontroller') {
                    checkout scm
                    container('go') {
                        sh "make test"
                        sh "./out/exposecontroller"
                    }
                }
                dir ('/home/jenkins/go/src/github.com/jenkins-x/exposecontroller/charts/exposecontroller') {
                    container('go') {
                        sh "helm init --client-only"

                        sh "make build"
                        sh "helm template ."
                    }
                }
            }
        }
    
        stage('Build and Release') {
            environment {
                GH_CREDS = credentials('jenkins-x-github')
            }
            when {
                branch 'master'
            }
            steps {
                dir ('/home/jenkins/go/src/github.com/jenkins-x/exposecontroller') {
                    checkout scm
                    container('go') {
                        sh "echo \$(jx-release-version) > version/VERSION"
                        sh "git add version/VERSION"
                        sh "git commit -m 'release \$(cat version/VERSION)'"

                        sh "GITHUB_ACCESS_TOKEN=$GH_CREDS_PSW make release"
                    }
                }
                dir ('/home/jenkins/exposecontroller') {
                    checkout scm
                    container('jx-base') {
                        // ensure we're not on a detached head
                        sh "git checkout master"

                        // until we switch to the new kubernetes / jenkins credential implementation use git credentials store
                        sh "git config credential.helper store"

                        sh "git checkout master"
                        sh "helm init --client-only"


                        sh "make release"
                    }
                }
            }
        }
    }
}
