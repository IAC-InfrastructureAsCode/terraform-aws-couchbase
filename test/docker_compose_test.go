package test

import (
	"testing"
	"fmt"
	"path/filepath"
	"github.com/gruntwork-io/terratest/test-structure"
	terralog "github.com/gruntwork-io/terratest/log"
	"github.com/gruntwork-io/terratest/files"
	"log"
	"github.com/gruntwork-io/terratest/shell"
)

func TestUnitCouchbaseSingleClusterUbuntuInDocker(t *testing.T) {
	t.Parallel()
	testCouchbaseInDocker(t, "TestUnitCouchbaseSingleClusterUbuntuInDocker","couchbase-single-cluster", "ubuntu", 2, 8091, 4984)
}

func TestUnitCouchbaseMultiClusterUbuntuInDocker(t *testing.T) {
	t.Parallel()
	testCouchbaseInDocker(t, "TestUnitCouchbaseMultiClusterUbuntuInDocker", "couchbase-multi-cluster","ubuntu", 3,7091, 3984)
}

func testCouchbaseInDocker(t *testing.T, testName string, examplesFolderName string, osName string, clusterSize int, couchbaseWebConsolePort int, syncGatewayWebConsolePort int) {
	logger := terralog.NewLogger(testName)

	tmpRootDir, err := files.CopyTerraformFolderToTemp("../", testName)
	if err != nil {
		t.Fatal(err)
	}
	couchbaseAmiDir := filepath.Join(tmpRootDir, "examples", "couchbase-ami")
	couchbaseSingleClusterDockerDir := filepath.Join(tmpRootDir, "examples", examplesFolderName, "local-test")

	test_structure.RunTestStage("setup_image", logger, func() {
		buildCouchbaseWithPacker(t, logger, fmt.Sprintf("%s-docker", osName), "couchbase","us-east-1", couchbaseAmiDir)
	})

	test_structure.RunTestStage("setup_docker", logger, func() {
		startCouchbaseWithDockerCompose(t, couchbaseSingleClusterDockerDir, testName, logger)
	})

	defer test_structure.RunTestStage("teardown", logger, func() {
		getDockerComposeLogs(t, couchbaseSingleClusterDockerDir, testName, logger)
		stopCouchbaseWithDockerCompose(t, couchbaseSingleClusterDockerDir, testName, logger)
	})

	test_structure.RunTestStage("validation", logger, func() {
		consoleUrl := fmt.Sprintf("http://localhost:%d", couchbaseWebConsolePort)
		checkCouchbaseConsoleIsRunning(t, consoleUrl, logger)

		dataNodesUrl := fmt.Sprintf("http://%s:%s@localhost:%d", usernameForTest, passwordForTest, couchbaseWebConsolePort)
		checkCouchbaseClusterIsInitialized(t, dataNodesUrl, clusterSize, logger)
		checkCouchbaseDataNodesWorking(t, dataNodesUrl, logger)

		syncGatewayUrl := fmt.Sprintf("http://localhost:%d/mock-couchbase-asg", syncGatewayWebConsolePort)
		checkSyncGatewayWorking(t, syncGatewayUrl, logger)
	})
}

func startCouchbaseWithDockerCompose(t *testing.T, exampleDir string, testName string, logger *log.Logger) {
	runDockerCompose(t, exampleDir, testName, logger, "up", "-d")
}

func getDockerComposeLogs(t *testing.T, exampleDir string, testName string, logger *log.Logger) {
	logger.Printf("Fetching docker-compose logs:")
	runDockerCompose(t, exampleDir, testName, logger, "logs")
}

func stopCouchbaseWithDockerCompose(t *testing.T, exampleDir string, testName string, logger *log.Logger) {
	runDockerCompose(t, exampleDir, testName, logger, "down")
	runDockerCompose(t, exampleDir, testName, logger, "rm", "-f")
}

func runDockerCompose(t *testing.T, exampleDir string, testName string, logger *log.Logger, args ... string) {
	cmd := shell.Command{
		Command:    "docker-compose",
		// We append --project-name to ensure containers from multiple different tests using Docker Compose don't end
		// up in the same project and end up conflicting with each other.
		Args:       append([]string{"--project-name", testName}, args...),
		WorkingDir: exampleDir,
	}

	if err := shell.RunCommand(cmd, logger); err != nil {
		t.Fatalf("Failed to run docker-compose %v in %s: %v", args, exampleDir, err)
	}
}