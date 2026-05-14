import React, { useEffect, useState } from 'react';
import ReactApexChart from 'react-apexcharts';
import type { ApexOptions } from 'apexcharts';
import { getSimTrades, getSimPortfolios } from '../api';

interface Props {
  portfolioId: number;
}

interface ROIPoint {
  x: number; // timestamp ms
  y: number; // ROI %
}

const PortfolioROIChart: React.FC<Props> = ({ portfolioId }) => {
  const [roiPoints, setRoiPoints] = useState<ROIPoint[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!portfolioId) return;
    setLoading(true);
    setError('');

    const fetchData = async () => {
      try {
        const simData = await getSimPortfolios();
        const pfItem = simData.portfolios.find(i => i.portfolio?.id === portfolioId);
        if (!pfItem) { setRoiPoints([]); return; }

        const coins = pfItem.states.map(s => s.coin);
        const totalInitial = pfItem.states.reduce((sum, s) => sum + s.initial_capital, 0);

        // Fetch trades for all coins in parallel
        const tradesPerCoin: Record<string, ReturnType<typeof getSimTrades> extends Promise<infer T> ? T extends { trades: infer U } ? U : never : never> = {};
        const results = await Promise.all(
          coins.map(coin => getSimTrades(coin, portfolioId).then(r => ({ coin, trades: r.trades || [] })))
        );
        results.forEach(({ coin, trades }) => {
          tradesPerCoin[coin] = trades as typeof tradesPerCoin[string];
        });

        // Collect all trade events and sort by time
        interface TradeEvent {
          coin: string;
          cash_after: number;
          units_after: number;
          price: number;
          ts: number;
        }

        const events: TradeEvent[] = [];
        for (const { coin, trades } of results) {
          for (const t of trades) {
            events.push({
              coin,
              cash_after: t.cash_after,
              units_after: t.units_after,
              price: t.price,
              ts: new Date(t.executed_at).getTime(),
            });
          }
        }
        events.sort((a, b) => a.ts - b.ts);

        if (events.length === 0) { setRoiPoints([]); return; }

        // Track per-coin running state
        const coinState: Record<string, { cash: number; units: number; lastPrice: number; initial: number }> = {};
        pfItem.states.forEach(s => {
          coinState[s.coin] = { cash: s.initial_capital, units: 0, lastPrice: 0, initial: s.initial_capital };
        });

        const points: ROIPoint[] = [];
        for (const ev of events) {
          if (!coinState[ev.coin]) continue;
          coinState[ev.coin].cash = ev.cash_after;
          coinState[ev.coin].units = ev.units_after;
          coinState[ev.coin].lastPrice = ev.price;

          // Sum total portfolio value across all coins
          let totalValue = 0;
          for (const state of Object.values(coinState)) {
            totalValue += state.cash + state.units * state.lastPrice;
          }

          const roi = totalInitial > 0 ? (totalValue - totalInitial) / totalInitial * 100 : 0;
          points.push({ x: ev.ts, y: parseFloat(roi.toFixed(2)) });
        }

        setRoiPoints(points);
      } catch (err) {
        setError('ROI 데이터 로딩 실패');
        console.error(err);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [portfolioId]);

  if (loading) {
    return (
      <div style={{ height: 100, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#888', fontSize: 12 }}>
        ⏳ ROI 차트 로딩 중...
      </div>
    );
  }

  if (error) {
    return (
      <div style={{ height: 60, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ef5350', fontSize: 12 }}>
        {error}
      </div>
    );
  }

  if (roiPoints.length === 0) {
    return (
      <div style={{ height: 60, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#555', fontSize: 12, border: '1px solid #2a2a4a', borderRadius: 6, marginBottom: 10 }}>
        거래 내역이 없습니다
      </div>
    );
  }

  const lastRoi = roiPoints[roiPoints.length - 1]?.y ?? 0;
  const lineColor = lastRoi >= 0 ? '#26a69a' : '#ef5350';

  const options: ApexOptions = {
    theme: { mode: 'dark' },
    chart: {
      background: '#0f0f1f',
      foreColor: '#ccc',
      toolbar: { show: false },
      animations: { enabled: false },
      sparkline: { enabled: false },
    },
    stroke: { curve: 'smooth', width: 2, colors: [lineColor] },
    fill: {
      type: 'gradient',
      gradient: {
        shadeIntensity: 1,
        type: 'vertical',
        gradientToColors: ['#ef5350'],
        stops: [0, 100],
        opacityFrom: 0.3,
        opacityTo: 0.05,
      },
    },
    colors: [lineColor],
    annotations: {
      yaxis: [
        {
          y: 0,
          borderColor: '#ef5350',
          borderWidth: 1,
          strokeDashArray: 4,
        },
      ],
    },
    xaxis: {
      type: 'datetime',
      labels: { style: { colors: '#888', fontSize: '10px' }, datetimeUTC: false },
      axisBorder: { color: '#2a2a4a' },
      axisTicks: { color: '#2a2a4a' },
    },
    yaxis: {
      labels: {
        style: { colors: '#888', fontSize: '10px' },
        formatter: (v: number) => v.toFixed(1) + '%',
      },
    },
    grid: { borderColor: '#2a2a4a', padding: { left: 4, right: 4, top: 0, bottom: 0 } },
    tooltip: {
      theme: 'dark',
      x: { format: 'yyyy-MM-dd HH:mm' },
      y: { formatter: (v: number) => (v >= 0 ? '+' : '') + v.toFixed(2) + '%' },
    },
    legend: { show: false },
  };

  return (
    <div style={{ background: '#0f0f1f', border: '1px solid #2a2a4a', borderRadius: 6, padding: '6px 4px 0', marginBottom: 10 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 10px 2px' }}>
        <span style={{ fontSize: 10, color: '#666', textTransform: 'uppercase', letterSpacing: '0.08em' }}>누적 ROI</span>
        <span style={{ fontSize: 12, fontWeight: 700, color: lineColor }}>
          {lastRoi >= 0 ? '+' : ''}{lastRoi.toFixed(2)}%
        </span>
      </div>
      <ReactApexChart
        options={options}
        series={[{ name: 'ROI', data: roiPoints }]}
        type="area"
        height={140}
      />
    </div>
  );
};

export default PortfolioROIChart;
