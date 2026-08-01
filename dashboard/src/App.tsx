import { useEffect } from "react";
import { liveStream } from "./lib/stream";
import { useLiveMetrics } from "./hooks/useLiveMetrics";
import { useDLQ } from "./hooks/useDLQ";
import { useLogStream } from "./hooks/useLogStream";
import { Header } from "./components/Header";
import { HealthStrip } from "./components/HealthStrip";
import { MetricCards } from "./components/MetricCards";
import { Panel } from "./components/Panel";
import { ThroughputChart } from "./components/ThroughputChart";
import { LatencyChart } from "./components/LatencyChart";
import { DLQTable } from "./components/DLQTable";
import { LogFeed } from "./components/LogFeed";

export default function App() {
  const metrics = useLiveMetrics();
  const dlq = useDLQ();
  const logs = useLogStream();

  useEffect(() => {
    liveStream.start();
    return () => liveStream.close();
  }, []);

  return (
    <div className="flex h-screen flex-col gap-2 overflow-hidden bg-bg p-2 font-sans text-bright">
      <Header status={metrics.status} lastUpdate={metrics.lastUpdate} />
      <HealthStrip
        lag={metrics.latest?.consumer.lag_messages}
        lagSeries={metrics.lagSeries}
        es={metrics.latest?.consumer.elasticsearch}
        bufferFill={metrics.latest?.ingestion.buffer_fill}
        bufferCapacity={metrics.latest?.ingestion.buffer_capacity}
        inFlight={metrics.latest?.consumer.in_flight}
        retrying={metrics.latest?.consumer.retrying_total}
      />
      <MetricCards series={metrics.series} />

      <div className="grid h-[200px] shrink-0 grid-cols-3 gap-2">
        <Panel title="Throughput · last 2 min" className="col-span-2">
          <ThroughputChart series={metrics.series} />
        </Panel>
        <Panel title="Latency · p99 / p50">
          <LatencyChart series={metrics.series} />
        </Panel>
      </div>

      <div className="grid min-h-[280px] flex-1 grid-cols-5 gap-2">
        <Panel
          title="Dead-letter queue"
          className="col-span-2"
          right={<span className="font-mono text-[10px] text-faint">{dlq.entries.length} entries</span>}
        >
          <DLQTable entries={dlq.entries} />
        </Panel>
        <Panel
          title="Log stream"
          className="col-span-3"
          right={<span className="font-mono text-[10px] text-faint">{logs.logs.length} buffered</span>}
        >
          <LogFeed logs={logs.logs} services={logs.services} />
        </Panel>
      </div>
    </div>
  );
}
