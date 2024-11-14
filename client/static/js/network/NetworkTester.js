// NetworkTester.js
class NetworkTester {
	constructor() {
		this.connectionManager = new ConnectionManager();
		this.metricsManager = new MetricsManager();
		this.testStartTime = null;
		this.isRunning = false;
		this.lastProfile = null;
		
		this.connectionManager.onStateChange((state) => {
			this.handleConnectionStateChange(state);
		});

		this.connectionManager.onMetrics((metricsData) => {
			this.metricsManager.processPacketData(metricsData);
		});

		this.metricsManager.onMetricsUpdate((metrics) => {
			// Removed empty updateUI call
		});

		this.metricsManager.onMetricsReport((report) => {
			this.connectionManager.sendMetricsReport(report);
		});

		this.connectionManager.onTestComplete((result) => {
			this.handleTestComplete(result);
		});

		this.connectionManager.onProfileUpdate((data) => {
			this.onProfileUpdate(data);
		})
	}

	async startTest(hostAddress, useSsl = false) {
		if (this.isRunning) {
			console.warn('Test is already running');
			return;
		}

		this.isRunning = true;
		this.testStartTime = Date.now();
		this.resetManagers();
		this.showMetricsUI();

		try {
			// Detect network capabilities
			await this.metricsManager.detectNetworkCapabilities();

			// Connect to the server
			await this.connectionManager.connect(hostAddress, useSsl);
		} catch (error) {
			console.error('Failed to start test:', error);
			this.stopTest();
			throw error;
		}
	}

	handleTestComplete(result) {

		const finalProfile = result || this.lastProfile?.profile || 'No profile available';
		console.log('Test Complete! Final Profile:', result);
		
		const testCompleteElement = document.getElementById('testComplete');
		if (testCompleteElement) {
			testCompleteElement.textContent = `Test Complete! Final Profile: ${result}`;
			testCompleteElement.style.display = 'block';
		}

		this.stopTest();
	}

	onProfileUpdate(data) {
		this.lastProfile = data; 
		this.metricsManager.updateSendRate(data.send_rate)
	}

	stopTest() {
		if (!this.isRunning) return;
		console.log("closing test")

		this.isRunning = false;
		this.connectionManager.disconnect();
	}

	resetManagers() {
		this.metricsManager.reset();
	}

	handleConnectionStateChange(state) {
		const stateElement = document.getElementById('connectionState');
		if (stateElement) {
			switch (state) {
				case 'connecting':
					stateElement.textContent = 'Connecting...';
					break;
				case 'connected':
					stateElement.textContent = 'Connected, receiving test packets...';
					break;
				case 'disconnected':
					stateElement.textContent = 'Disconnected';
					this.stopTest();
					break;
				case 'failed':
					stateElement.textContent = 'Connection failed';
					this.stopTest();
					break;
				default:
					stateElement.textContent = state;
			}
		}
	}

	showMetricsUI() {
		const metricsDiv = document.getElementById('networkMetrics');
		if (metricsDiv) {
			metricsDiv.style.display = 'block';
		}
	}
}

// Initialize and add event listener
document.addEventListener('DOMContentLoaded', () => {
	const tester = new NetworkTester();
	const startButton = document.getElementById('startNetworkTest');
	
	if (startButton) {
		startButton.addEventListener('click', async () => {
			const hostAddress = document.getElementById('hostAddress').value;
			const useSsl = document.getElementById('useSsl').checked;
			
			try {
				await tester.startTest(hostAddress, useSsl);
			} catch (error) {
				console.error('Test failed:', error);
			}
		});
	}
});