pipeline {
  agent {
    docker {
      image 'registry.atc/base/build:latest'
      args '--tmpfs /tmp:uid=666 -v /var/run/docker.sock:/var/run/docker.sock --group-add 997'
      alwaysPull true
    }
  }
  parameters {
    string(name: 'BASE_ID', defaultValue: '', description: 'Base commit ID')
    string(name: 'DIFF_ID', defaultValue: '', description: 'Phabricator diff ID')
    string(name: 'REV_ID', defaultValue: '', description: 'Phabricator Revision ID (Dxxx)')
    string(name: 'PHID', defaultValue: '', description: 'Phabricator target operation ID')
  }
  stages {
    stage('Clone') {
      steps {
        script {
          if (env.BASE_ID == '') { 
            env.BASE_ID = "refs/tags/phabricator/diff/${env.DIFF_ID}" 
          }
        }
        checkout([$class: 'GitSCM',
          branches: [[ name: "${env.BASE_ID}" ]],
          userRemoteConfigs: [[
            url: 'github.com:ataccama/aws-usage.git',
            credentialsId: 'github-aws-usage-deploy-key'
          ]]
        ])
      }
    }
    stage('Build image') {
      steps {
        script {
          docker.withRegistry('https://registry.atc/', 'registry.atc') {
            sh "git log -1 > .version"
            img = docker.build("aws-usage")
            if (env.REV_ID == 'D') {
              img.push("latest")
            } else {
              img.push("${env.REV_ID}")
            }
          }
        }
      }
    }
  }
  post {
    always {
      script {
        if (currentBuild.result == null) {
          currentBuild.result = currentBuild.currentResult
        }
        step([$class: 'PhabricatorNotifier', commentOnSuccess: true, commentWithConsoleLinkOnFailure: true])
        cleanWs()
      }
    }
  }
}
