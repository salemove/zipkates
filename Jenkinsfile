@Library('pipeline-lib') _
@Library('cve-monitor') __

def MAIN_BRANCH = 'master'
def DOCKER_PROJECT_NAME = 'salemove/zipkates'
def DOCKER_REGISTRY_URL = 'https://registry.hub.docker.com'
def DOCKER_REGISTRY_CREDENTIALS_ID = '6992a9de-fab7-4932-9907-3aba4a70c4c0'

withResultReporting(slackChannel: '#tm-inf') {
  inDockerAgent(containers: [
    interactiveContainer(name: 'go', image: 'golang:1.13'),
    imageScanner.container()
  ]) {
    checkout(scm)
    stage('Run tests') {
      ansiColor('xterm') {
        container('go') {
          sh('''
            go mod download
            go test ./...
          ''')
        }
      }
    }
    def version = shEval('git describe --tags --always --dirty || echo "unknown"')
    def image
    stage('Build docker image') {
      ansiColor('xterm') {
        image = docker.build("${DOCKER_PROJECT_NAME}:${version}")
      }
    }
    stage('Scan docker image') {
      imageScanner.scan(image)
    }
    stage('Publish docker image') {
      docker.withRegistry(DOCKER_REGISTRY_URL, DOCKER_REGISTRY_CREDENTIALS_ID) {
        if (BRANCH_NAME == MAIN_BRANCH) {
          echo("Publishing docker image ${image.imageName()} with tag ${version} and latest")
          image.push("${version}")
          image.push("latest")
        } else {
          echo("${BRANCH_NAME} is not the master branch. Not publishing the docker image.")
        }
      }
    }
  }
}

def shEval(cmd) {
  sh(returnStdout: true, script: cmd).trim()
}
