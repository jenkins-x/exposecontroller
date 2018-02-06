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
                        sh "make"
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
                CHARTMUSEUM_CREDS = credentials('jenkins-x-chartmuseum')
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
                dir ('/home/jenkins/go/src/github.com/jenkins-x/exposecontroller/charts/exposecontroller') {
                    container('go') {
                        sh "helm init --client-only"
                        sh "make release"
                    }
                }
            }
        }
    }
}
