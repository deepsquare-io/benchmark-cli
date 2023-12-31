package benchmark_test

import (
	"bytes"
	"context"
	"log"
	"testing"
	"text/template"

	"github.com/squarefactory/benchmark-api/benchmark"
	"github.com/squarefactory/benchmark-api/mocks"
	"github.com/squarefactory/benchmark-api/scheduler"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

var (
	JobName = "HPL-Benchmark"
	admin   = "root"
)

type ServiceTestSuite struct {
	suite.Suite
	scheduler *mocks.Scheduler
	impl      *benchmark.Benchmark
}

func (suite *ServiceTestSuite) BeforeTest(suiteName, testName string) {
	suite.scheduler = mocks.NewScheduler(suite.T())

	suite.impl = &benchmark.Benchmark{
		SlurmClient: suite.scheduler,
		Dat: benchmark.DATParams{
			ProblemSize:  "",
			P:            2,
			Q:            3,
			NProblemSize: 1,
			NBlockSize:   1,
			BlockSize:    "64",
		},
		Sbatch: benchmark.SBATCHParams{
			ContainerPath: "/etc/hpl-benchmark/hpc-benchmarks:hpl.sqsh",
			Workspace:     "/etc/hpl-benchmark",
			Node:          1,
			NtasksPerNode: 2,
			GpusPerNode:   2,
			CpusPerTasks:  8,
			CpuAffinity:   "6-7:2-3",
			GpuAffinity:   "0:1",
		},
	}

}

func (suite *ServiceTestSuite) TestRun() {

	// Arrange
	files := benchmark.BenchmarkFile{
		DatFile:    "testdatfile",
		SbatchFile: "testsbatchfile",
	}

	expectedSubmitRequest := &scheduler.SubmitRequest{
		Name: JobName,
		User: admin,
		Body: "testsbatchfile",
	}

	suite.scheduler.On(
		"Submit",
		mock.Anything,
		expectedSubmitRequest,
	).Return("test submit response", nil)

	// Act
	err := suite.impl.Run(context.Background(), &files)

	// Assert
	suite.NoError(err)
	suite.scheduler.AssertExpectations(suite.T())
}

func (suite *ServiceTestSuite) TestGenerateDAT() {
	// Arrange
	expectedTemplate := `HPLinpack benchmark input file
Innovative Computing Laboratory, University of Tennessee
HPL.out      output file name (if any)
6            device out (6=stdout,7=stderr,file)
1 # of problems sizes (N)
  Ns
1   # of NBs
64    NBs
0            PMAP process mapping (0=Row-,1=Column-major)
1            # of process grids (P x Q)
2     Ps
3      Qs
16.0         threshold
1            # of panel fact
2            PFACTs (0=left, 1=Crout, 2=Right)
1            # of recursive stopping criterium
4            NBMINs (>= 1)
1            # of panels in recursion
2            NDIVs
1            # of recursive panel fact.
1            RFACTs (0=left, 1=Crout, 2=Right)
1            # of broadcast
1            BCASTs (0=1rg,1=1rM,2=2rg,3=2rM,4=Lng,5=LnM)
1            # of lookahead depth
1            DEPTHs (>=0)
2            SWAP (0=bin-exch,1=long,2=mix)
64           swapping threshold
1            L1 in (0=transposed,1=no-transposed) form
0            U  in (0=transposed,1=no-transposed) form
1            Equilibration (0=no,1=yes)
8            memory alignment in double (> 0)
##### This line (no. 32) is ignored (it serves as a separator). ######
0                               Number of additional problem sizes for PTRANS
1200 10000 30000                values of N
0                               number of additional blocking sizes for PTRANS
40 9 8 13 13 20 16 32 64        values of NB
`

	// Act
	result, err := suite.impl.GenerateDAT()

	// Assert
	suite.NoError(err)

	tmpl, err := template.New("jobTemplate").Parse(expectedTemplate)
	if err != nil {
		log.Fatalf("Error parsing template: %s", err)
	}

	var expectedBuffer bytes.Buffer
	err = tmpl.Execute(&expectedBuffer, suite.impl.Dat)
	if err != nil {
		log.Fatalf("Error executing template: %s", err)
	}

	suite.Equal(expectedBuffer.String(), result)
}

func (suite *ServiceTestSuite) TestGenerateSBATCH() {
	// Arrange
	expectedTemplate := `#!/bin/sh

#SBATCH -N 1
#SBATCH --ntasks-per-node=2
#SBATCH --gpus-per-node=2
#SBATCH --mem=0
#SBATCH --cpus-per-task=8
#SBATCH --gpus-per-task=1

export PMIX_MCA_pml=ob1
export PMIX_MCA_btl=vader,self,tcp
export OMPI_MCA_pml=ob1
export OMPI_MCA_btl=vader,self,tcp

srun  --mpi=pmix_v4 --cpu-bind=none --gpu-bind=none --container-image="/etc/hpl-benchmark/hpc-benchmarks:hpl.sqsh" \
  --container-mounts="/etc/hpl-benchmark/hpl.dat:/test.dat" sh -c 'sed -Ei "s/:1//g" ./hpl.sh && ./hpl.sh --xhpl-ai --cpu-affinity 6-7:2-3 --cpu-cores-per-rank 8 --gpu-affinity 0:1 --dat "/test.dat"'
`

	// Act
	result, err := suite.impl.GenerateMultiNodeSBATCH()

	// Assert
	suite.NoError(err)

	tmpl, err := template.New("jobTemplate").Parse(expectedTemplate)
	if err != nil {
		log.Fatalf("Error parsing template: %s", err)
	}

	var expectedBuffer bytes.Buffer
	err = tmpl.Execute(&expectedBuffer, suite.impl.Dat)
	if err != nil {
		log.Fatalf("Error executing template: %s", err)
	}

	suite.Equal(expectedBuffer.String(), result)
}

func (suite *ServiceTestSuite) TestCalculateProcessGrid() {
	// Arrange
	P, Q := 2, 2

	suite.scheduler.On(
		"FindGPUPerNode",
		mock.Anything,
	).Return(2)

	// Act
	err := suite.impl.CalculateProcessGrid(context.Background())

	// Assert
	suite.NoError(err)
	suite.scheduler.AssertExpectations(suite.T())
	suite.Equal(P, suite.impl.Dat.P)
	suite.Equal(Q, suite.impl.Dat.Q)
}

func (suite *ServiceTestSuite) TestCalculateProblemSize() {
	// Arrange
	expectedMem := "95000 96000 97000 98000 100000 101000 102000 103000 105000 106000 "
	suite.scheduler.On(
		"FindMemPerNode",
		mock.Anything,
	).Return(128460, nil)

	// Act
	err := suite.impl.CalculateProblemSize(context.Background())

	// Assert
	suite.NoError(err)
	suite.scheduler.AssertExpectations(suite.T())
	suite.Equal(expectedMem, suite.impl.Dat.ProblemSize)
}

func (suite *ServiceTestSuite) TestCalculateAffinity() {

	expectedCpu := "6-7:2-3"
	expectedGpu := "0:1"

	affinity := `0 6-7
1 2-3`

	suite.scheduler.On(
		"FindCPUAffinity",
		mock.Anything,
	).Return(affinity, nil)

	err := suite.impl.CalculateAffinity(context.Background())

	suite.NoError(err)
	suite.scheduler.AssertExpectations(suite.T())
	suite.Equal(expectedCpu, suite.impl.Sbatch.CpuAffinity)
	suite.Equal(expectedGpu, suite.impl.Sbatch.GpuAffinity)
}

func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, &ServiceTestSuite{})
}
