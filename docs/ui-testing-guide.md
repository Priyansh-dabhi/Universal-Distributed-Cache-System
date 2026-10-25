# Interactive Web Studio — Testing & Verification Guide

This guide provides a step-by-step walkthrough for testing every module of the **Universal Distributed Cache Interactive Web Studio**.

It explains **what is happening under the hood**, **what to click**, **what to observe**, and **what expected results look like**.

---

## Table of Contents
1. [Core Concepts & Dual-Mode Engine](#1-core-concepts--dual-mode-engine)
2. [Quickstart: Launching the Studio](#2-quickstart-launching-the-studio)
3. [Module 1: Consistent Hash Ring Visualizer (`#/ring`)](#3-module-1-consistent-hash-ring-visualizer)
4. [Module 2: CRUD Studio & Node Inspector (`#/playground`)](#4-module-2-crud-studio--node-inspector)
5. [Module 3: Eviction Policy Visualizer (`#/eviction`)](#5-module-3-eviction-policy-visualizer)
6. [Module 4: Telemetry & Traffic Generator (`#/metrics`)](#6-module-4-telemetry--traffic-generator)
7. [Module 5: Interactive Architecture Guide (`#/learn`)](#7-module-5-interactive-architecture-guide)
8. [Testing Live Cluster Mode (Go Backend)](#8-testing-live-cluster-mode-go-backend)
9. [Automated Lint & Typecheck Verification](#9-automated-lint--typecheck-verification)
10. [Troubleshooting & FAQs](#10-troubleshooting--faqs)

---

## 1. Core Concepts & Dual-Mode Engine

The UI is built with a **Dual-Mode Engine**:

```text
               ┌───────────────────────────────┐
               │    Web UI (Port 3000)         │
               └──────────────┬────────────────┘
                              │
               Mode Toggle in Header Bar
               /                             \
     [Web Simulator Mode]           [Live Cluster Mode]
              │                              │
  Runs 100% in Browser           Communicates via HTTP REST
  • TypeScript FNV-1a Hasher     • Router at :9000
  • Consistent Hash Ring         • Node-1 at :8001 (LRU)
  • LRU, LFU, 2Q Engines         • Node-2 at :8002 (LFU)
  • Zero installation required   • Node-3 at :8003 (2Q)
```

* **Web Simulator Mode (Default)**: You don't need Go, compilers, or server processes running. Everything executes right inside your browser with high performance.
* **Live Cluster Mode**: Talks directly to real Go background servers over HTTP REST.

---

## 2. Quickstart: Launching the Studio

1. Open your terminal in the `ui` folder:
   ```bash
   cd ui
   npm run dev
   ```
2. Open your browser and navigate to:
   ```text
   http://localhost:3000
   ```
3. You will see the **Distributed Cache Studio** with a dark-cyber theme, a top status header, and 5 interactive tabs:
   * **Hash Ring**: Circular consistent hashing visualizer.
   * **CRUD Studio**: Key-value query console & cluster memory inspector.
   * **Eviction Lab**: LRU, LFU, 2Q queue visualizers & table scan demo.
   * **Metrics**: Real-time hit rate gauge, charts, and synthetic traffic runners.
   * **Learn Guide**: Searchable architecture concept textbook.

---

## 3. Module 1: Consistent Hash Ring Visualizer

**Route:** `#/ring`  
**Purpose:** Understand how consistent hashing routes arbitrary string keys to distributed cache nodes using virtual tokens, and why it prevents the "cache stampede" of modulo hashing.

### What is happening under the hood?
1. The key is hashed using **32-bit FNV-1a** (`uint32`).
2. The hash ring represents the entire range $[0, 2^{32} - 1]$ mapped to a $360^\circ$ circle.
3. Each physical node owns **100 virtual node tokens** scattered uniformly across the ring.
4. The router searches clockwise using binary search ($O(\log R)$) to find the first token whose position $\ge \text{hash}(\text{key})$.
5. That token's parent node stores and serves the key.

---

### Step-by-Step Tests to Perform:

#### Test 1.1: Route a Key & Inspect the Math Breakdown
1. In the **Key Routing Console**, click on one of the quick preset buttons: `user:101`, `cart:9981`, or `session:xyz`.
2. Alternatively, type any custom string (e.g. `order:alpha`) and click **Route Key**.
3. **What to observe:**
   * An animated **orange data packet** sweeps clockwise around the circular ring and lands on the designated node.
   * The selected node's virtual tokens glow and pulse.
   * The **Route Result Card** displays a 4-step breakdown:
     - **Step 1 (FNV-1a Hash)**: Exact decimal and hexadecimal 32-bit hash.
     - **Step 2 (Ring Angle)**: Angle in degrees (`(hash / 2^32) * 360°`).
     - **Step 3 (Clockwise Search)**: Token identifier found via binary search (e.g. `node-2#47`).
     - **Step 4 (Target Node)**: Physical node assigned to handle the request.

#### Test 1.2: Add a Chaos Node & Observe Token Scattering
1. Look at the **Cluster Chaos & Topology** panel on the left.
2. In the "Add New Node" section, type an ID (e.g. `node-4`) and click **Add Node to Ring**.
3. **What to observe:**
   * A toast notification announces that `node-4` was registered.
   * 100 new colored tokens scatter instantly across the $360^\circ$ ring.
   * The total token counter increases from 300 to 400.

#### Test 1.3: Run the Key Redistribution Test (Consistent Hashing vs Modulo)
1. Scroll down to the **Key Redistribution Benchmark** panel.
2. Click **Run 100-Key Redistribution Test**.
3. **What to observe:**
   * The visualizer simulates adding an extra node to the cluster and tests where 100 sample keys route.
   * **Consistent Hashing**: Only **~15% to 25%** of keys change nodes ($1/(N+1)$ expected).
   * **Traditional Modulo Hashing**: **~75%** of keys change nodes ($N/(N+1)$)!
   * This proves why consistent hashing prevents massive cache churn when servers scale up or down.

#### Test 1.4: Remove a Node & Check Clockwise Traffic Handoff
1. In the Active Nodes list, click the **Remove** button (red trash can icon) next to `node-4`.
2. **What to observe:**
   * `node-4`'s tokens smoothly disappear from the ring.
   * Total tokens decrease back to 300.
   * Rerouting the same key now smoothly advances clockwise to the next available node.

---

## 4. Module 2: CRUD Studio & Node Inspector

**Route:** `#/playground`  
**Purpose:** Test read/write/delete operations, verify transparent proxy routing, observe TTL expiration, and inspect resident keys per node.

---

### Step-by-Step Tests to Perform:

#### Test 2.1: Store a Key with Value (`PUT`)
1. In the **Request Bar**, ensure the HTTP method is set to **PUT**.
2. Set Key: `user:101`.
3. Set Value: `{"name": "Alice", "role": "Engineer"}`.
4. Leave TTL at `No TTL (Infinite)`.
5. Click **Send Request**.
6. **What to observe:**
   * A success toast confirms the key was stored.
   * The **Response Panel** displays:
     - Status: `200 OK`
     - Proxy Hop Trace: `Client ──► Router (:9000) ──► Target Node`
     - Formatted JSON response payload: `{"message": "stored", "evicted": false}`.
   * In the **Cluster Nodes** grid below, the target node's memory capacity bar increments.
   * A new row appears in the **Request History** table with method `PUT`, key, status, and round-trip latency in milliseconds.

#### Test 2.2: Retrieve the Stored Key (`GET`)
1. Switch the method dropdown to **GET**.
2. Keep the Key as `user:101` and click **Send Request**.
3. **What to observe:**
   * Status: `200 OK`.
   * A green **HIT** badge appears in the hop trace.
   * The Response Panel prints the exact JSON value previously saved.
   * In the Request History table, the operation is logged as a **HIT**.

#### Test 2.3: Test Cache Miss (`404 Not Found`)
1. Switch method to **GET**.
2. Type an un-stored key: `user:non_existent`.
3. Click **Send Request**.
4. **What to observe:**
   * Status: `404 Not Found`.
   * A rose **MISS** badge is displayed.
   * The response payload explains `{"error": "key not found"}`.

#### Test 2.4: Test TTL Expiration (Lazy Eviction)
1. Switch method to **PUT**.
2. Key: `temp:code`.
3. Value: `SecretPasscode123`.
4. In the **TTL Dropdown**, select `5 Seconds`.
5. Click **Send Request** (Status `200 OK`).
6. **Immediately** send a `GET temp:code` -> Status `200 OK` (Hit).
7. **Wait 6 seconds**, then send `GET temp:code` again.
8. **What to observe:**
   * Status: `404 Not Found`.
   * Error message: `"key expired"`.
   * This proves the engine's lazy expiration mechanism works without background thread overhead.

#### Test 2.5: Slide-Over Node Inspector Drawer
1. In the **Cluster Nodes** grid, locate one of the nodes and click **Inspect Node**.
2. **What to observe:**
   * A slide-over drawer animates in from the right edge.
   * Displays the node's:
     - Physical port & address
     - Active eviction policy (LRU / LFU / 2Q)
     - Current memory utilization percentage
     - List of currently resident keys, values, and expiration countdowns.
3. Click the close button or anywhere outside to dismiss the drawer.

---

## 5. Module 3: Eviction Policy Visualizer

**Route:** `#/eviction`  
**Purpose:** Visually observe how LRU, LFU, and 2Q handle memory pressure, cache updates, and malicious scan attacks.

---

### Step-by-Step Tests to Perform:

#### Test 3.1: Test LRU (Least Recently Used)
1. Click the **LRU (Least Recently Used)** tab.
2. Notice the queue structure: **HEAD (Most Recently Used)** on the left $\longleftrightarrow$ **TAIL (Least Recently Used)** on the right.
3. Click the quick button **Fill to Max (5/5)**.
4. Five items populate the linked list: `item:1` through `item:5`.
5. In the Key input, type `item:1` (which is currently near the tail) and click **Get**.
6. **What to observe:**
   * `item:1` highlights in green and smoothly animates across the screen straight to the **HEAD** (MRU) position!
   * The remaining elements shift right.
7. Now add a 6th item: Key `item:6`, Value `V6`, and click **Set**.
8. **What to observe:**
   * The item at the **TAIL** turns red with an eviction tag and drops out of the list.
   * `item:6` is inserted at the **HEAD**.
   * The eviction counter increases by 1.

#### Test 3.2: Test LFU (Least Frequently Used)
1. Click the **LFU (Least Frequently Used)** tab.
2. Items are grouped into distinct horizontal frequency tiers: `Freq: 1`, `Freq: 2`, `Freq: 3+`.
3. Click **Fill to Max (5/5)**. All 5 items start in the `Freq: 1` tier.
4. Now query `item:2` two times by clicking **Get** twice.
5. **What to observe:**
   * `item:2` moves from the `Freq: 1` tier up into the `Freq: 3+` tier!
   * Its access count badge shows `3 accesses`.
6. Insert a new item (`item:new`).
7. **What to observe:**
   * The engine finds the lowest non-empty frequency tier (`Freq: 1`).
   * It evicts the oldest item inside that lowest tier (using LRU tie-breaking).
   * High-frequency items in higher tiers remain protected.

#### Test 3.3: Test 2Q (Two-Queue Scan-Resistant Engine)
1. Click the **2Q (Two-Queue)** tab.
2. Observe the two distinct sub-queues:
   * **Queue $A1$ (FIFO Probation, 25% capacity)**: Gateway queue for new, unproven keys.
   * **Queue $A_m$ (LRU Main, 75% capacity)**: High-priority queue for frequently referenced keys.
3. Insert a new key: `data:x` and click **Set**.
4. **What to observe:**
   * `data:x` enters into **Queue $A1$ (Probation)**. It has NOT entered Main yet.
5. Now access `data:x` by typing its key and clicking **Get**.
6. **What to observe:**
   * A promotion animation triggers!
   * `data:x` is removed from probation and promoted into **Queue $A_m$ (Main LRU)**!
   * Future accesses to `data:x` will now refresh its position within the Main queue.

#### Test 3.4: Run the Table Scan Attack Demonstration
1. Scroll down to the **Scan Attack Pollution Simulator** at the bottom of the Eviction page.
2. Read the explanation: A database scan or web scraper reading hundreds of single-use keys can completely wipe out standard LRU caches.
3. Click **Simulate Table Scan Attack (10 sequential keys)**.
4. **What to observe:**
   * The simulator fires 10 rapid single-use keys (`scan:1` to `scan:10`).
   * **LRU Result**: 100% of your hot cached keys were flushed out and replaced with useless scan keys!
   * **2Q Result**: Your hot keys inside $A_m$ remained **completely safe**! The scan keys merely rotated through the small $A1$ probation queue and were discarded without polluting the main cache.

---

## 6. Module 4: Telemetry & Traffic Generator

**Route:** `#/metrics`  
**Purpose:** Monitor real-time cluster health, hit rates, request throughput, and benchmark performance under realistic synthetic access patterns.

---

### Step-by-Step Tests to Perform:

#### Test 4.1: Inspect the Real-Time Telemetry Cards
1. Look at the top metric cards:
   * **Total Requests**
   * **Cache Hits** (emerald glow)
   * **Cache Misses** (rose glow)
   * **Evictions** (amber glow)
   * **Sets / Writes** (cyan glow)
   * **Expired Keys** (violet glow)
2. In the center, view the **Hit Rate Circular Gauge**:
   * Displays the dynamic hit percentage computed as $\frac{\text{Hits}}{\text{Hits} + \text{Misses}} \times 100\%$.
   * Features a smooth animated SVG stroke with color-coded grading (Green $>75\%$, Cyan $>50\%$, Amber $<50\%$).

#### Test 4.2: Run a Zipfian (Hot-Key / 80-20 Rule) Workload
1. Under the **Synthetic Workload Runner** panel, select the **Zipfian (Hot-Key)** tab.
2. Select **100 Requests**.
3. Click **Start Workload**.
4. **What to observe:**
   * A progress bar tracks the batches in real time.
   * ~80% of accesses target the top 20% most popular keys.
   * Watch the **Hit Rate Gauge** climb up to **75% – 90%**!
   * The **Throughput Area Chart** shows requests per second surging.
   * The **Latency Breakdown** chart shows fast response times with almost all `200 OK` status codes.

#### Test 4.3: Run a Uniform Random Workload
1. Switch the workload tab to **Uniform Random**.
2. Select **100 Requests** and click **Start Workload**.
3. **What to observe:**
   * Requests are distributed uniformly across an open range of keys.
   * Watch the **Hit Rate Gauge** drop down to **15% – 30%** (as expected for cold, random data).
   * The **Cache Misses** counter increments rapidly.

#### Test 4.4: Emergency Stop
1. Start a **250 Requests** workload.
2. Halfway through, click the red **Stop / Abort** button.
3. **What to observe:**
   * The runner immediately ceases traffic.
   * An alert confirms the test was canceled without freezing the UI or leaving hung promises.

---

## 7. Module 5: Interactive Architecture Guide

**Route:** `#/learn`  
**Purpose:** Explore the underlying computer science concepts and see direct references to the Go source code.

---

### Step-by-Step Tests to Perform:

#### Test 5.1: Instant Keyword Search
1. In the search box, type: `FNV`.
2. **What to observe:**
   * Cards instantly filter to **Topic 1.1: FNV-1a Hashing Algorithm**.
   * Highlights the mathematical formula ($h = (h \times 16777619) \oplus c$), why 32-bit was selected, and links to `internal/router/hasher.go`.

#### Test 5.2: Filter by Architecture Tier
1. In the sidebar, click on **Tier 2: Cache Eviction Mechanics**.
2. **What to observe:**
   * Displays in-depth cards for:
     - **LRU (Doubly-Linked List + Sentinel Nodes)**
     - **LFU (Multi-Tier Frequency Buckets with $O(1)$ updates)**
     - **2Q (Scan-Resistant Dual-Queue Algorithm)**
   * Each card features syntax-highlighted Go code snippets directly from `internal/cache/`.

#### Test 5.3: Deep-Link Jump to Live Visualizers
1. On any concept card (e.g. *LRU Architecture*), click the cyan button **Try in Eviction Visualizer** or **Try in Hash Ring**.
2. **What to observe:**
   * The UI smoothly routes you straight into that interactive visualizer tab with the relevant feature pre-selected.

---

## 8. Testing Live Cluster Mode (Go Backend)

To test the Web Studio connected to real Go processes:

### Step 1: Start the 3 Go Cache Server Nodes
Open three separate terminal windows:

```powershell
# Terminal 1: Cache Node 1 (port 8001, LRU)
go run ./cmd/cache-server --id node-1 --port 8001 --capacity 100 --policy lru

# Terminal 2: Cache Node 2 (port 8002, LFU)
go run ./cmd/cache-server --id node-2 --port 8002 --capacity 100 --policy lfu

# Terminal 3: Cache Node 3 (port 8003, 2Q)
go run ./cmd/cache-server --id node-3 --port 8003 --capacity 100 --policy 2q
```

### Step 2: Start the Go Reverse-Proxy Router
Open a fourth terminal:

```powershell
# Terminal 4: Distributed Router (port 9000, 100 virtual replicas)
go run ./cmd/router --port 9000 --nodes node-1=http://localhost:8001,node-2=http://localhost:8002,node-3=http://localhost:8003 --replicas 100
```

### Step 3: Switch the Web UI to Live Mode
1. In the Web UI header, locate the mode toggle in the top right.
2. Click **Live Cluster**.
3. **What to observe:**
   * The status badge turns **emerald green**: `● Live Cluster: Connected`.
   * Open your browser's Developer Tools (`F12` -> **Network** tab).
   * Perform a `PUT` or `GET` in the CRUD Studio.
   * You will see real HTTP requests to `/api/cache/:key` returning JSON payloads directly from your running Go router and cache servers!

---

## 9. Automated Lint & Typecheck Verification

Run these commands in the `ui` folder to verify strict code quality:

```bash
cd ui

# 1. Run both Oxlint and ESLint
npm run lint
# Expected Output: Found 0 warnings and 0 errors.

# 2. Run TypeScript compiler across all project references
npm run typecheck
# Expected Output: Exits cleanly with code 0.

# 3. Build optimized production bundle
npm run build
# Expected Output: Transforms 2982 modules and outputs chunks in <1s.
```

---

## 10. Troubleshooting & FAQs

### Q: Why does the Live Mode indicator say "Disconnected"?
* **Cause**: The Go router is not running on port 9000, or a firewall is blocking local HTTP traffic.
* **Fix**: Ensure `go run ./cmd/router ...` is actively running in a terminal. Click the gear icon in the UI header and verify the address is `http://127.0.0.1:9000`. Click **Test Connection**.

### Q: Does Web Simulator mode affect my Go cluster?
* **No**. In Web Simulator mode, all operations are isolated to in-memory TypeScript structures in your browser. It will never write to or read from the Go servers.

### Q: How do I reset all simulated data in Web Simulator mode?
* Simply refresh your browser tab (`F5` or `Ctrl+R`). The simulation engine re-initializes to a pristine default 3-node cluster.

### Q: Can I run the UI on a phone or tablet?
* **Yes**. The layout is completely responsive. The circular hash ring automatically scales, and the tab bar scrolls horizontally to fit any mobile viewport.
