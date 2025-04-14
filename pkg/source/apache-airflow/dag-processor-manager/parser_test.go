package apacheairflow

import (
	"testing"

	"github.com/GoogleCloudPlatform/khi/pkg/model"
	"github.com/stretchr/testify/assert"

	_ "github.com/GoogleCloudPlatform/khi/internal/testflags"
)

func TestDagProcessor(t *testing.T) {

	testCases := []struct {
		name     string
		text     string
		expected *model.DagFileProcessorStats
	}{
		{
			"Real Data(with 7)",
			"/home/airflow/gcs/dags/airflow_monitoring.py  19517  0.08s             1           0  0.51s           2024-05-08T02:44:13",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"0.08s",
				"1",
				"0",
			),
		},
		{
			"minimum",
			"/home/airflow/gcs/dags/airflow_monitoring.py 1 0",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"",
				"1",
				"0",
			),
		},
		{
			"4 with PID",
			"/home/airflow/gcs/dags/airflow_monitoring.py  18419                   1           0",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"",
				"1",
				"0",
			),
		},
		{
			"4 with RUNTIME",
			"/home/airflow/gcs/dags/airflow_monitoring.py 2.58s 1 0",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"2.58s",
				"1",
				"0",
			),
		},
		{
			"5 with PID and RUNTIME",
			"/home/airflow/gcs/dags/airflow_monitoring.py 19517 0.08s 1 0",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"0.08s",
				"1",
				"0",
			),
		},
		{
			"5 with LAST_*",
			"/home/airflow/gcs/dags/airflow_monitoring.py 1 0  0.51s 2024-05-08T02:44:13",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"",
				"1",
				"0",
			),
		},
		{
			"6 with RUNTIME",
			"/home/airflow/gcs/dags/airflow_monitoring.py 0.08s 1 0  0.51s 2024-05-08T02:44:13",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"0.08s",
				"1",
				"0",
			),
		},
		{
			"6 with PID and RUNTIME and LAST_*",
			"/home/airflow/gcs/dags/airflow_monitoring.py 19517  0.08s 1 0  0.51s",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"0.08s",
				"1",
				"0",
			),
		},
		{
			"6 with PID and LAST_*",
			"/home/airflow/gcs/dags/airflow_monitoring.py  19517 1 0  0.51s 2024-05-08T02:44:13",
			model.NewDagFileProcessorStats(
				"/home/airflow/gcs/dags/airflow_monitoring.py",
				"",
				"1",
				"0",
			),
		},
	}

	p := &AirflowDagProcessorParser{
		dagFilePath: "/home/airflow/gcs/dags/",
	}

	for _, test := range testCases {
		t.Run("Test_"+test.name, func(t *testing.T) {
			stats := p.fromLogEntity(test.text)
			assert.NotNil(t, stats)
			assert.Equal(t, test.expected, stats)
		})
	}

}
