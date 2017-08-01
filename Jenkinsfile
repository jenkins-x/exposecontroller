#!/usr/bin/groovy
@Library('github.com/fabric8io/fabric8-pipeline-library@master')
def dummy
goNode{
  dockerNode{
    ws{
      if (env.BRANCH_NAME.startsWith('PR-')) {
        goCI{
          githubOrganisation = 'fabric8io'
          dockerOrganisation = 'fabric8'
          project = 'exposecontroller'
          makeTarget = 'clean test cross'
        }
      } else if (env.BRANCH_NAME.equals('master')) {
        def v = goRelease{
          githubOrganisation = 'fabric8io'
          dockerOrganisation = 'fabric8'
          project = 'exposecontroller'
        }

        stage ('Update downstream dependencies') {
          updateDownstreamDependencies(v)
        }
      }
    }
  }
}

def updateDownstreamDependencies(v) {
  pushPomPropertyChangePR {
    propertyName = 'exposecontroller.version'
    projects = [
            'fabric8io/fabric8-devops',
            'fabric8io/fabric8-platform',
            'funktionio/funktion-platform'
    ]
    version = v
  }
}