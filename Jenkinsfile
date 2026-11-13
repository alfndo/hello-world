// Pipeline hello-world: Build -> Verify (kandidat) -> Deploy (+rollback) -> Promote, plus pembersihan image.
// Pemicu (Poll SCM) tetap diatur di job Jenkins, bukan di file ini.

// Uji respons container lewat sidecar curl yang berbagi network container target
// (image akhir memakai "scratch", jadi tidak ada curl atau shell di dalamnya).
def smoke(String container, String expected = 'hello world') {
    sh """
        sleep 2
        OUT=\$(docker run --rm --network container:${container} curlimages/curl -fsS http://localhost:8080)
        echo "Response: \$OUT"
        echo "\$OUT" | grep -q "${expected}"
    """
}

pipeline {
  agent any

  options {
    timestamps()
    disableConcurrentBuilds()
    timeout(time: 10, unit: 'MINUTES')
  }

  parameters {
    booleanParam(name: 'SIMULATE_DEPLOY_FAILURE', defaultValue: false,
                 description: 'Khusus demo rollback: paksa smoke test tahap Deploy gagal')
  }

  environment {
    IMAGE = 'kai/hello-world'
    NAME  = 'hello-world'
    PORT  = '8091'   // 8080 = pevita-web, 8090 = Jenkins
    KEEP  = '5'      // jumlah image bernomor build yang disimpan
  }

  stages {
    stage('Build') {
      steps {
        sh 'docker build -t $IMAGE:$BUILD_NUMBER .'
      }
    }

    // Uji versi baru di container kandidat (tanpa port publik). Container live tidak disentuh.
    stage('Verify') {
      steps {
        sh 'docker rm -f $NAME-candidate || true'
        sh 'docker run -d --name $NAME-candidate $IMAGE:$BUILD_NUMBER'
        script { smoke("${env.NAME}-candidate") }
      }
      post {
        failure { sh 'docker logs $NAME-candidate || true' }
        always  { sh 'docker rm -f $NAME-candidate || true' }
      }
    }

    // Ganti container live, uji lagi, dan rollback ke versi stable jika gagal.
    stage('Deploy') {
      steps {
        sh 'docker rm -f $NAME || true'
        sh 'docker run -d --name $NAME --restart unless-stopped -p $PORT:8080 $IMAGE:$BUILD_NUMBER'
        script { smoke(env.NAME, params.SIMULATE_DEPLOY_FAILURE ? 'TEKS_TIDAK_ADA' : 'hello world') }
      }
      post {
        failure {
          sh '''
            echo "Deploy gagal, rollback ke versi stable"
            docker logs $NAME || true
            docker rm -f $NAME || true
            if docker image inspect $IMAGE:stable >/dev/null 2>&1; then
              docker run -d --name $NAME --restart unless-stopped -p $PORT:8080 $IMAGE:stable
            else
              echo "Belum ada versi stable untuk rollback"
            fi
          '''
        }
      }
    }

    // Tandai versi ini "stable" hanya jika semua tahap sebelumnya lolos.
    stage('Promote') {
      steps {
        sh 'docker tag $IMAGE:$BUILD_NUMBER $IMAGE:stable'
      }
    }
  }

  post {
    success {
      // Simpan $KEEP image terbaru (dan tag stable), hapus sisanya. Hanya menyentuh kai/hello-world.
      sh '''
        docker images $IMAGE --format '{{.Tag}}' | grep -E '^[0-9]+$' | sort -rn | tail -n +$((KEEP + 1)) | while read t; do
          docker rmi $IMAGE:$t || true
        done
        docker image prune -f
      '''
    }
  }
}
