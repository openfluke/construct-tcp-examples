package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/websocket/v2"
)

// --- Live Monitoring System ---

type LiveState struct {
	BodyCount      int                 `json:"body_count"`
	Metrics        PerformanceResponse `json:"metrics"`
	Timestamp      string              `json:"timestamp"`
	SpawningActive bool                `json:"spawning_active"`
	StopReason     string              `json:"stop_reason"`
}

var (
	clients         = make(map[*websocket.Conn]bool)
	clientsMu       sync.Mutex
	broadcast       = make(chan LiveState)
	spawningEnabled = true
	controlMu       sync.Mutex
)

func RunLoadTest() {
	fmt.Println("🚀 Starting Test 13: WEB-SOCKET LIVE MONITORING 🚀")

	// 1. Connect to Construct Server
	conn, err := net.Dial("tcp", "localhost:17000")
	if err != nil {
		fmt.Printf("❌ Failed to connect to Construct Server: %v\n", err)
		return
	}
	defer conn.Close()

	// 2. Start Fiber Server
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})
	app.Use(logger.New())

	app.Get("/", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(dashboardHTML)
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		clientsMu.Lock()
		clients[c] = true
		clientsMu.Unlock()

		defer func() {
			clientsMu.Lock()
			delete(clients, c)
			clientsMu.Unlock()
			c.Close()
		}()

		for {
			var msg struct {
				Action string `json:"action"`
			}
			if err := c.ReadJSON(&msg); err != nil {
				break
			}
			if msg.Action == "stop" {
				controlMu.Lock()
				spawningEnabled = false
				controlMu.Unlock()
				fmt.Println("\n🛑 MANUAL STOP TRIGGERED BY USER")
			}
		}
	}))

	go func() {
		log.Fatal(app.Listen(":3000"))
	}()

	fmt.Println("🌐 Dashboard live at: http://localhost:3000")

	// 3. Broadcast Hub
	go func() {
		for state := range broadcast {
			clientsMu.Lock()
			for client := range clients {
				err := client.WriteJSON(state)
				if err != nil {
					client.Close()
					delete(clients, client)
				}
			}
			clientsMu.Unlock()
		}
	}()

	// 4. Main Spawning & Querying Loop
	writePacket(conn, []byte(`{"type":"query_state"}`))
	buf := make([]byte, 32768)
	n, _ := conn.Read(buf)
	var state StateResponse
	json.Unmarshal(buf[:n], &state)

	spawnCenter := Vector3{state.PlayerPos[0], state.PlayerPos[1] + 10, state.PlayerPos[2]}
	bodyCount := 0
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		// Stop spawning if FPS drops below 20
		stopReason := ""
		controlMu.Lock()
		active := spawningEnabled
		controlMu.Unlock()

		if active {
			// Spawn 10 rigid bodies
			for i := 0; i < 10; i++ {
				id := fmt.Sprintf("live_box_%d", bodyCount)
				offset := Vector3{
					(float32(bodyCount%10) - 5) * 1.5,
					float32(bodyCount/10) * 1.0,
					0,
				}
				pos := Add(spawnCenter, offset)
				createBox(conn, id, pos)
				bodyCount++
			}
		}

		// Query performance
		writePacket(conn, []byte(`{"type":"query_performance"}`))
		header := make([]byte, 4)
		_, err := conn.Read(header)
		if err != nil {
			continue
		}
		length := int(binary.LittleEndian.Uint32(header))
		pbuf := make([]byte, length)
		_, err = conn.Read(pbuf)
		if err != nil {
			continue
		}

		var perf PerformanceResponse
		if err := json.Unmarshal(pbuf, &perf); err == nil {
			// Check if we should stop. Sensitivity increased.
			controlMu.Lock()
			if spawningEnabled && (perf.EngineFPS < 20 || perf.TimeFPS < 20) && bodyCount > 10 {
				spawningEnabled = false
				stopReason = fmt.Sprintf("FPS LIMIT REACHED (Engine: %.1f, Smoothed: %.1f)", perf.EngineFPS, perf.TimeFPS)
				fmt.Printf("\n🛑 STOPPING SPAWN: FPS dropped below 20 at %d bodies\n", bodyCount)
			}
			active = spawningEnabled
			controlMu.Unlock()

			// Broadcast to Web Dashboard
			broadcast <- LiveState{
				BodyCount:      bodyCount,
				Metrics:        perf,
				Timestamp:      time.Now().Format("15:04:05.000"),
				SpawningActive: active,
				StopReason:     stopReason,
			}
		}

		status := "RUNNING"
		if !spawningEnabled {
			status = "HALTED"
		}
		fmt.Printf("\r📦 Bodies: %d | FPS: %.1f | RAM: %.1fMB | Status: %s", bodyCount, perf.EngineFPS, perf.MemoryStatic, status)
	}
}

func createBox(conn net.Conn, id string, pos Vector3) {
	createReq := ConstructRequest{
		Type:        "create_construct",
		ConstructID: id,
		Parts: []Part{
			{
				ID:     "core",
				Type:   "box",
				Size:   Vector3{1, 1, 1},
				Pos:    pos,
				Color:  Vector3{0.3, 0.8, 1.0},
				Locked: false,
			},
		},
	}
	data, _ := json.Marshal(createReq)
	writePacket(conn, data)
}

const dashboardHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>PrimeCraft Pro Monitor</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        :root { --bg: #020617; --card: #0f172a; --border: #1e293b; --text: #f8fafc; --muted: #94a3b8; --accent: #38bdf8; --danger: #ef4444; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: var(--bg); color: var(--text); margin: 0; padding: 20px; }
        .header { display: flex; justify-content: space-between; align-items: center; padding-bottom: 20px; border-bottom: 1px solid var(--border); margin-bottom: 30px; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(500px, 1fr)); gap: 20px; }
        .card { background: var(--card); padding: 20px; border-radius: 12px; border: 1px solid var(--border); box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1); position: relative; }
        .metric-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 15px; margin-bottom: 20px; }
        .mini-card { background: #1e293b; padding: 12px; border-radius: 8px; border: 1px solid var(--border); }
        .mini-val { font-size: 1.5rem; font-weight: bold; color: var(--accent); }
        .mini-label { font-size: 0.8rem; color: var(--muted); text-transform: uppercase; letter-spacing: 0.05em; }
        canvas { width: 100% !important; height: 350px !important; }
        .status-pill { padding: 6px 16px; background: #064e3b; color: #4ade80; border-radius: 9999px; font-size: 0.875rem; display: flex; align-items: center; gap: 8px; }
        .status-pill.halted { background: #450a0a; color: #f87171; }
        .status-dot { width: 8px; height: 8px; border-radius: 50%; }
        .status-pill.halted .status-dot { background: #f87171; }
        .status-pill.live .status-dot { background: #4ade80; }
        h2 { font-size: 1.25rem; margin-top: 0; color: #cbd5e1; display: flex; align-items: center; gap: 10px; font-weight: bold; }
        
        .btn { padding: 10px 20px; border-radius: 8px; border: none; font-weight: bold; cursor: pointer; transition: 0.2s; }
        .btn-primary { background: var(--accent); color: var(--bg); }
        .btn-danger { background: var(--danger); color: white; }
        
        #report-overlay { display: none; position: fixed; inset: 0; background: var(--bg); z-index: 1000; overflow-y: auto; padding: 40px; }
        .report-header { text-align: center; margin-bottom: 40px; }
        .analysis-box { background: #1e293b; padding: 30px; border-radius: 12px; border-left: 6px solid var(--accent); margin-bottom: 30px; }
        
        @media print {
            .no-print { display: none !important; }
            #report-overlay { position: static; display: block !important; padding: 0; }
            body { background: white !important; color: black !important; }
            .card { border: 1px solid #ccc !important; box-shadow: none !important; page-break-inside: avoid; }
            .mini-card { border: 1px solid #eee !important; }
            .metric-label { color: #666 !important; }
            .mini-val { color: black !important; }
        }
    </style>
</head>
<body>
    <div class="header no-print">
        <div>
            <h1 style="margin:0; font-size: 1.5rem;">PrimeCraft <span style="color:var(--accent)">PRO</span> Monitor</h1>
            <p style="margin:5px 0 0; color:var(--muted)">Exhaustive Server Stress Testing Dashboard</p>
        </div>
        <div style="display:flex; gap:10px; align-items:center;">
            <button class="btn btn-danger" id="btn-stop">FORCE STOP</button>
            <button class="btn btn-primary" id="btn-report" style="display:none;">Generate Report</button>
            <div id="status-pill" class="status-pill live"><div class="status-dot"></div> <span id="status-text">Live Streaming</span></div>
        </div>
    </div>
    
    <div class="metric-grid no-print" style="grid-template-columns: repeat(4, 1fr);">
        <div class="mini-card">
            <div class="mini-val" id="val-bodies">0</div>
            <div class="mini-label">Rigid Bodies</div>
        </div>
        <div class="mini-card">
            <div class="mini-val" id="val-fps">0.0</div>
            <div class="mini-label">Engine FPS</div>
        </div>
        <div class="mini-card">
            <div class="mini-val" id="val-ram">0.0 MB</div>
            <div class="mini-label">Static RAM</div>
        </div>
        <div class="mini-card">
            <div class="mini-val" id="val-draw">0</div>
            <div class="mini-label">Draw Calls</div>
        </div>
    </div>

    <div class="grid no-print">
        <div class="card"><h2>STATISTICS &bull; Performance</h2><canvas id="chart-perf"></canvas></div>
        <div class="card"><h2>TIMINGS &bull; Frame Latency</h2><canvas id="chart-time"></canvas></div>
        <div class="card"><h2>MEMORY &bull; Allocation</h2><canvas id="chart-mem"></canvas></div>
        <div class="card"><h2>SCENE &bull; Complexity</h2><canvas id="chart-complexity"></canvas></div>
        <div class="card"><h2>PHYSICS &bull; 3D Engine</h2><canvas id="chart-physics"></canvas></div>
        <div class="card"><h2>FULL LOG &bull; All Metrics</h2><div class="metric-grid" id="all-metrics-container"></div></div>
    </div>

    <!-- Final Report Overlay -->
    <div id="report-overlay">
        <div class="report-header">
            <h1 style="font-size: 3rem; margin-bottom: 0;">PERFORMANCE AUDIT REPORT</h1>
            <p id="report-date" style="color:var(--muted)"></p>
            <button class="btn btn-primary no-print" onclick="window.print()">Save as PDF / Print</button>
            <button class="btn no-print" onclick="document.getElementById('report-overlay').style.display='none'">Back to Dashboard</button>
        </div>

        <div class="analysis-box">
            <h2 style="color:var(--accent); margin-bottom:15px;">EXECUTIVE SUMMARY</h2>
            <div id="analysis-text" style="line-height:1.6; font-size:1.1rem;">
                <!-- Analysis generated by JS -->
            </div>
        </div>

        <div class="grid" id="report-charts">
            <!-- Snapshots of charts will be cloned here -->
        </div>

        <div class="card" style="margin-top:30px;">
            <h2 style="margin-bottom:20px;">LATEST PULSE MATRICS</h2>
            <div class="metric-grid" id="report-metrics-grid"></div>
        </div>
    </div>

    <script>
        const createChart = (id, datasets) => {
            return new Chart(document.getElementById(id).getContext('2d'), {
                type: 'line',
                data: { labels: [], datasets: datasets.map(d => ({
                    ...d, 
                    borderWidth: 2, 
                    pointRadius: 0, 
                    tension: 0.3, 
                    fill: false 
                }))},
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    animation: false,
                    scales: {
                        y: { grid: { color: '#1e293b' }, ticks: { color: '#cbd5e1', font: { size: 12, weight: 'bold', family: 'Arial' }, padding: 10 } },
                        x: { grid: { display: false }, ticks: { color: '#cbd5e1', font: { size: 12, weight: 'bold', family: 'Arial' }, maxRotation: 0, autoSkip: true, maxTicksLimit: 10, padding: 10 } }
                    },
                    plugins: { legend: { position: 'top', labels: { color: '#f8fafc', boxWidth: 15, font: { size: 13, weight: 'bold', family: 'Arial' }, padding: 20 } } }
                }
            });
        };

        const charts = {
            perf: createChart('chart-perf', [{ label: 'Engine FPS', data: [], borderColor: '#38bdf8' }, { label: 'Smoothed FPS', data: [], borderColor: '#818cf8', borderDash: [5, 5] }]),
            time: createChart('chart-time', [{ label: 'Total Process', data: [], borderColor: '#fbbf24' }, { label: 'Physics MS', data: [], borderColor: '#f87171' }, { label: 'Nav MS', data: [], borderColor: '#34d399' }]),
            mem: createChart('chart-mem', [{ label: 'Static RAM', data: [], borderColor: '#38bdf8' }, { label: 'Texture MB', data: [], borderColor: '#c084fc' }, { label: 'Video MB', data: [], borderColor: '#fb7185' }]),
            complexity: createChart('chart-complexity', [{ label: 'Nodes', data: [], borderColor: '#94a3b8' }, { label: 'Objects', data: [], borderColor: '#64748b' }, { label: 'Draw Calls', data: [], borderColor: '#38bdf8' }]),
            physics: createChart('chart-physics', [{ label: 'Active Obj', data: [], borderColor: '#f87171' }, { label: 'Collisions', data: [], borderColor: '#fca5a5' }, { label: 'Islands', data: [], borderColor: '#fbbf24' }])
        };

        let history = [];
        const ws = new WebSocket('ws://' + window.location.host + '/ws');
        
        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const m = data.metrics;
            const b = data.body_count;
            history.push(data);

            updateTopMetrics(data);
            updateAllMetricsList(m, 'all-metrics-container');

            if (!data.spawning_active) {
                const pill = document.getElementById('status-pill');
                pill.className = 'status-pill halted';
                document.getElementById('status-text').innerText = 'HALTED: ' + data.stop_reason;
                document.getElementById('btn-report').style.display = 'block';
            }

            // Update Charts
            const label = b;
            Object.values(charts).forEach(c => {
                c.data.labels.push(label);
                if (c.data.labels.length > 100) {
                    c.data.labels.shift();
                    c.data.datasets.forEach(d => d.data.shift());
                }
            });

            charts.perf.data.datasets[0].data.push(m.engine_fps);
            charts.perf.data.datasets[1].data.push(m.time_fps);
            charts.time.data.datasets[0].data.push(m.time_process);
            charts.time.data.datasets[1].data.push(m.time_physics_process);
            charts.time.data.datasets[2].data.push(m.time_navigation_process);
            charts.mem.data.datasets[0].data.push(m.memory_static);
            charts.mem.data.datasets[1].data.push(m.render_texture_mem_used);
            charts.mem.data.datasets[2].data.push(m.render_video_mem_used);
            charts.complexity.data.datasets[0].data.push(m.object_node_count);
            charts.complexity.data.datasets[1].data.push(m.object_count);
            charts.complexity.data.datasets[2].data.push(m.render_total_draw_calls_in_frame);
            charts.physics.data.datasets[0].data.push(m.physics_3d_active_objects);
            charts.physics.data.datasets[1].data.push(m.physics_3d_collision_pairs);
            charts.physics.data.datasets[2].data.push(m.physics_3d_island_count);

            Object.values(charts).forEach(c => c.update('none'));
        };

        function updateTopMetrics(data) {
            document.getElementById('val-bodies').innerText = data.body_count;
            document.getElementById('val-fps').innerText = data.metrics.engine_fps.toFixed(1);
            document.getElementById('val-ram').innerText = data.metrics.memory_static.toFixed(1) + ' MB';
            document.getElementById('val-draw').innerText = data.metrics.render_total_draw_calls_in_frame;
        }

        function updateAllMetricsList(m, containerId) {
            const container = document.getElementById(containerId);
            container.innerHTML = '';
            const skip = ["type", "engine_fps", "memory_static"];
            Object.entries(m).forEach(([key, val]) => {
                if (skip.some(s => key.includes(s))) return;
                const div = document.createElement('div');
                div.className = 'mini-card';
                div.innerHTML = '<div class="mini-label" style="font-size:0.6rem">' + key.replace(/_/g, ' ').toUpperCase() + '</div>' +
                                '<div class="mini-val" style="font-size:0.9rem">' + (typeof val === 'number' ? val.toFixed(2) : val) + '</div>';
                container.appendChild(div);
            });
        }

        document.getElementById('btn-report').onclick = () => {
            const overlay = document.getElementById('report-overlay');
            overlay.style.display = 'block';
            document.getElementById('report-date').innerText = 'Generated at: ' + new Date().toLocaleString();
            
            const last = history[history.length - 1];
            const m = last.metrics;

            // Simple Analysis
            let bottleneck = "None";
            if (m.time_physics_process > m.time_process * 0.7) bottleneck = "Physics Engine (High Island Count)";
            else if (m.render_total_draw_calls_in_frame > 2000) bottleneck = "Rendering (Draw Call Limit)";
            else if (m.engine_fps < 25) bottleneck = "General CPU Saturation";

            document.getElementById('analysis-text').innerHTML = 
                'The stress test concluded at <b>' + last.body_count + '</b> rigid bodies. ' +
                'Spawning was automatically halted due to: <b>' + last.stop_reason + '</b>.<br><br>' +
                '<b>CRITICAL LIMITS:</b><br>' +
                '- Max Stable Bodies: ' + Math.floor(last.body_count * 0.8) + ' (estimated at 60fps)<br>' +
                '- Peak Memory Usage: ' + m.memory_static_max.toFixed(2) + ' MB<br>' +
                '- Identified Bottleneck: <b style="color:var(--danger)">' + bottleneck + '</b><br><br>' +
                '<b>RECOMMENDATION:</b> Consider optimizing mesh merging to reduce draw calls or increasing physics sub-stepping if islands exceed ' + m.physics_3d_island_count + '.';

            updateAllMetricsList(m, 'report-metrics-grid');
            
            // Generate report chart snapshots
            const chartsGrid = document.getElementById('report-charts');
            chartsGrid.innerHTML = '';
            
            // Wait a tick for overlay to render
            setTimeout(() => {
                Object.entries(charts).forEach(([key, chart]) => {
                    const card = chart.canvas.closest('.card');
                    const clone = card.cloneNode(true);
                    const newCanvas = document.createElement('canvas');
                    newCanvas.style.height = '350px';
                    
                    // Clear the cloned canvas/content and insert our new one
                    clone.innerHTML = '<h2>' + card.querySelector('h2').innerHTML + '</h2>';
                    clone.appendChild(newCanvas);
                    chartsGrid.appendChild(clone);
                    
                    new Chart(newCanvas.getContext('2d'), {
                        type: 'line',
                        data: JSON.parse(JSON.stringify(chart.data)),
                        options: { 
                            ...chart.options, 
                            animation: false,
                            responsive: true,
                            maintainAspectRatio: false
                        }
                    });
                });
            }, 100);
        };

        document.getElementById('btn-stop').onclick = () => {
            ws.send(JSON.stringify({ action: 'stop' }));
            document.getElementById('btn-stop').style.display = 'none';
        };
    </script>
</body>
</html>
`
