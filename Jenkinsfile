pipeline {
  agent any

  options {
    timestamps()
    disableConcurrentBuilds()
  }

  environment {
    IMAGE = 'kai/hello-world'
    NAME  = 'hello-world'
    PORT  = '8091'   // 8080 = pevita-web, 8090 = Jenkins
  }

  stages {
    stage('Build') {
      steps {
        sh 'docker build -t $IMAGE:$BUILD_NUMBER -t $IMAGE:latest .'
      }
    }

    stage('Deploy') {
      steps {
        sh '''
          docker rm -f $NAME || true
          docker run -d --name $NAME --restart unless-stopped -p $PORT:8080 $IMAGE:latest
        '''
      }
    }

    stage('Smoke test') {
      steps {
        sh '''
          sleep 2
          docker run --rm --network container:$NAME curlimages/curl -fsS http://localhost:8080 | grep -q "hello world"
        '''
      }
    }
  }

  post {
    failure {
      sh 'docker logs $NAME || true'
    }
  }
}
