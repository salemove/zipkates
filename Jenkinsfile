@Library('pipeline-lib') _
@Library('cve-monitor') __

def MAIN_BRANCH = 'master'
def DOCKER_PROJECT_NAME = 'salemove/zipkates'
def DOCKER_REGISTRY_URL = 'https://registry.hub.docker.com'
def DOCKER_REGISTRY_CREDENTIALS_ID = '6992a9de-fab7-4932-9907-3aba4a70c4c0'

withResultReporting(slackChannel: '#tm-inf') {
  inDockerAgent(
    containers: [
      interactiveContainer(name: 'go', image: 'golang:1.13'),
      imageScanner.container()
    ],
    yaml: '''\
      apiVersion: v1
      kind: Pod
      spec:
        containers:
        - name: kind-cluster
          image: jieyu/kind-cluster-buster:v0.1.0
          stdin: true
          tty: true
          command:
          - /bin/bash
          args:
          - -c
          - cd / && exec /entrypoint.sh /bin/bash
          env:
          - name: API_SERVER_ADDRESS
            valueFrom:
              fieldRef:
                fieldPath: status.podIP
          volumeMounts:
          - mountPath: /var/lib/docker
            name: varlibdocker
          - mountPath: /lib/modules
            name: libmodules
            readOnly: true
          securityContext:
            privileged: true
          ports:
          - containerPort: 30001
            name: api-server-port
            protocol: TCP
          readinessProbe:
            failureThreshold: 15
            httpGet:
              path: /healthz
              port: api-server-port
              scheme: HTTPS
            initialDelaySeconds: 120
            periodSeconds: 20
            successThreshold: 1
            timeoutSeconds: 1
        volumes:
        - name: varlibdocker
          emptyDir: {}
        - name: libmodules
          hostPath:
            path: /lib/modules
    '''.stripIndent(),
    slaveConnectTimeout: 300
  ) {
    checkout([
      $class: 'GitSCM',
      branches: scm.branches,
      doGenerateSubmoduleConfigurations: scm.doGenerateSubmoduleConfigurations,
      extensions: scm.extensions + [[$class: 'CloneOption', noTags: false]],
      userRemoteConfigs: scm.userRemoteConfigs
    ])
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
        echo("Publishing docker image ${image.imageName()} with tag ${version}")
        image.push("${version}")
        if (BRANCH_NAME == MAIN_BRANCH) {
          echo("Also publishing with tag latest")
          image.push("latest")
        }
      }
    }
    stage('Run integration test') {
      ansiColor('xterm') {
        container('kind-cluster') {
          sh("""\
            #!/bin/bash
            set -euo pipefail

            # Start a zipkin instance with the sidecar and wait for it to be ready
            sed 's|image:.*zipkates:build|image: ${DOCKER_PROJECT_NAME}:${version}|' test-setup.yml |
              kubectl apply --wait=true -f-
            kubectl -n test-zipkin rollout status deploy/zipkin

            # Run the test
            kubectl apply --wait=true -f test.yml
            kubectl -n test-service wait --for=condition=complete --timeout=300s job/zipkin-client
          """.stripIndent())
        }
      }
    }
  }
}

def shEval(cmd) {
  sh(returnStdout: true, script: cmd).trim()
}
