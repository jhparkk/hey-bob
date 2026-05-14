import React, { useEffect, useState, useCallback, useRef } from 'react';
import ReactApexChart from 'react-apexcharts';
import type { ApexOptions, ApexAxisChartSeries } from 'apexcharts';
import { getPrices, getLivePrice } from '../api';
import type { PriceData, LivePriceData } from '../api';

type Coin = 'BTC' | 'ETH' | 'SOL';

interface ChartPageProps {
  isActive?: boolean;
}

interface IndicatorState {
  ma7: boolean; ma20: boolean; ma50: boolean;
  ema9: boolean; ema21: boolean;
  bb: boolean; rsi: boolean; macd: boolean; adx: boolean;
}

const PERIODS = [
  { label: '1W', value: '1w' },
  { label: '2W', value: '2w' },
  { label: '3W', value: '3w' },
  { label: '1M', value: '1m' },
  { label: '1Y', value: '1y' },
  { label: '직접설정', value: 'custom' },
];

function fmtMoney(n: number): string {
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

function formatBigNum(n: number): string {
  if (n >= 1e9) return '$' + (n / 1e9).toFixed(2) + 'B';
  if (n >= 1e6) return '$' + (n / 1e6).toFixed(2) + 'M';
  if (n >= 1e3) return '$' + (n / 1e3).toFixed(2) + 'K';
  return '$' + n.toFixed(2);
}

const darkBase: ApexOptions = {
  theme: { mode: 'dark' },
  chart: {
    background: '#1a1a2e',
    foreColor: '#ccc',
    toolbar: { show: true, tools: { download: false } },
    zoom: { enabled: true },
    animations: { enabled: false },
  },
  grid: {
    borderColor: '#2a2a4a',
  },
  tooltip: {
    theme: 'dark',
  },
  xaxis: {
    type: 'datetime',
    labels: { style: { colors: '#888' }, datetimeUTC: false },
    axisBorder: { color: '#2a2a4a' },
    axisTicks: { color: '#2a2a4a' },
  },
};

const ChartPage: React.FC<ChartPageProps> = ({ isActive }) => {
  const [coin, setCoin] = useState<Coin>('BTC');
  const [period, setPeriod] = useState<string>('2w');
  const [fromDate, setFromDate] = useState('');
  const [toDate, setToDate] = useState('');
  const [data, setData] = useState<PriceData[]>([]);
  const [liveData, setLiveData] = useState<LivePriceData | null>(null);
  const [indicators, setIndicators] = useState<IndicatorState>({
    ma7: false, ma20: false, ma50: false,
    ema9: false, ema21: false,
    bb: false, rsi: false, macd: false, adx: false,
  });

  const liveTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // display:none → display:flex 전환 시 ApexCharts 치수 재계산 강제
  useEffect(() => {
    if (isActive) {
      window.dispatchEvent(new Event('resize'));
    }
  }, [isActive]);

  // ── Data loading ──────────────────────────────────────────────────────────

  const loadData = useCallback(async (c: Coin, p: string, from?: string, to?: string) => {
    try {
      const json = await getPrices(c, p !== 'custom' ? p : undefined, from, to);
      if (!json.success || !json.data?.length) { setData([]); return; }
      const sorted = [...json.data].sort((a, b) => a.date < b.date ? -1 : 1);
      setData(sorted);
    } catch (err) {
      console.error('loadData error:', err);
    }
  }, []);

  const fetchLive = useCallback(async (c: Coin) => {
    try {
      const json = await getLivePrice(c);
      if (json.success) setLiveData(json.data);
    } catch (_) {}
  }, []);

  // live 데이터로 오늘 캔들 합성 — data 마지막 행이 오늘이면 업데이트, 아니면 추가
  useEffect(() => {
    if (!liveData || !data.length) return;
    const now = new Date();
    const seoulDate = new Date(now.toLocaleString('en-US', { timeZone: 'Asia/Seoul' }));
    const today = `${seoulDate.getFullYear()}-${String(seoulDate.getMonth() + 1).padStart(2, '0')}-${String(seoulDate.getDate()).padStart(2, '0')}`;
    const openPrice = liveData.last_price - (liveData.price_change || 0);
    const todayCandle: PriceData = {
      id: -1, coin: liveData.coin || '', date: today,
      open: openPrice, high: liveData.high_price, low: liveData.low_price, close: liveData.last_price,
      volume: liveData.volume,
      ma7: 0, ma20: 0, ma50: 0, ema9: 0, ema21: 0,
      rsi14: 0, macd: 0, macd_signal: 0,
      bb_upper: 0, bb_middle: 0, bb_lower: 0, adx14: 0,
    };
    setData(prev => {
      if (!prev.length) return prev;
      const last = prev[prev.length - 1];
      if (last.date === today) {
        // 오늘 행 업데이트
        return [...prev.slice(0, -1), { ...last, ...todayCandle }];
      } else if (last.date < today) {
        // 오늘 행 추가
        return [...prev, todayCandle];
      }
      return prev;
    });
  }, [liveData]); // eslint-disable-line react-hooks/exhaustive-deps

  // 초기 로딩
  useEffect(() => {
    loadData('BTC', '2w');
    fetchLive('BTC');
    liveTimerRef.current = setInterval(() => fetchLive(coin), 5000);
    return () => { if (liveTimerRef.current) clearInterval(liveTimerRef.current); };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // live 타이머 갱신
  useEffect(() => {
    if (liveTimerRef.current) clearInterval(liveTimerRef.current);
    liveTimerRef.current = setInterval(() => fetchLive(coin), 5000);
    return () => { if (liveTimerRef.current) clearInterval(liveTimerRef.current); };
  }, [coin, fetchLive]);

  // ── Handlers ──────────────────────────────────────────────────────────────

  const handleCoinChange = (c: Coin) => {
    setCoin(c);
    if (period === 'custom') {
      loadData(c, 'custom', fromDate, toDate);
    } else {
      loadData(c, period);
    }
    fetchLive(c);
  };

  const handlePeriodChange = (p: string) => {
    setPeriod(p);
    if (p !== 'custom') {
      loadData(coin, p);
    }
  };

  const handleCustomApply = () => {
    loadData(coin, 'custom', fromDate, toDate);
  };

  const toggleIndicator = (key: keyof IndicatorState) => {
    setIndicators(prev => ({ ...prev, [key]: !prev[key] }));
  };

  // ── Chart series ─────────────────────────────────────────────────────────

  const mainSeries = React.useMemo((): ApexAxisChartSeries => {
    if (!data.length) return [];

    const series: ApexAxisChartSeries = [
      {
        name: 'OHLC',
        type: 'candlestick',
        data: data.map(d => ({
          x: new Date(d.date).getTime(),
          y: [d.open, d.high, d.low, d.close] as [number, number, number, number],
        })),
      },
    ];

    if (indicators.ma7) {
      series.push({
        name: 'MA7',
        type: 'line',
        data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.ma7 || null })) as { x: number; y: number | null }[],
      });
    }
    if (indicators.ma20) {
      series.push({
        name: 'MA20',
        type: 'line',
        data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.ma20 || null })) as { x: number; y: number | null }[],
      });
    }
    if (indicators.ma50) {
      series.push({
        name: 'MA50',
        type: 'line',
        data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.ma50 || null })) as { x: number; y: number | null }[],
      });
    }
    if (indicators.ema9) {
      series.push({
        name: 'EMA9',
        type: 'line',
        data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.ema9 || null })) as { x: number; y: number | null }[],
      });
    }
    if (indicators.ema21) {
      series.push({
        name: 'EMA21',
        type: 'line',
        data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.ema21 || null })) as { x: number; y: number | null }[],
      });
    }
    if (indicators.bb) {
      series.push({
        name: 'BB Upper',
        type: 'line',
        data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.bb_upper || null })) as { x: number; y: number | null }[],
      });
      series.push({
        name: 'BB Middle',
        type: 'line',
        data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.bb_middle || null })) as { x: number; y: number | null }[],
      });
      series.push({
        name: 'BB Lower',
        type: 'line',
        data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.bb_lower || null })) as { x: number; y: number | null }[],
      });
    }

    return series;
  }, [data, indicators]);

  const mainOptions = React.useMemo((): ApexOptions => {
    const strokeColors: string[] = ['transparent']; // candlestick
    const strokeWidths: number[] = [1];
    const strokeDashes: number[] = [0];
    const lineNames = [];

    if (indicators.ma7)  { strokeColors.push('#26a69a'); strokeWidths.push(1.5); strokeDashes.push(0); lineNames.push('MA7'); }
    if (indicators.ma20) { strokeColors.push('#f4d03f'); strokeWidths.push(1.5); strokeDashes.push(0); lineNames.push('MA20'); }
    if (indicators.ma50) { strokeColors.push('#e74c3c'); strokeWidths.push(1.5); strokeDashes.push(0); lineNames.push('MA50'); }
    if (indicators.ema9) { strokeColors.push('#00bcd4'); strokeWidths.push(1.5); strokeDashes.push(4); lineNames.push('EMA9'); }
    if (indicators.ema21){ strokeColors.push('#ff9800'); strokeWidths.push(1.5); strokeDashes.push(4); lineNames.push('EMA21'); }
    if (indicators.bb)   {
      strokeColors.push('#7b68ee', '#7b68ee', '#7b68ee');
      strokeWidths.push(1, 1, 1);
      strokeDashes.push(3, 0, 3);
      lineNames.push('BB Upper', 'BB Middle', 'BB Lower');
    }

    return {
      ...darkBase,
      chart: {
        ...darkBase.chart,
        id: 'main-chart',
        group: 'crypto',
        type: 'candlestick',
        height: 460,
      },
      plotOptions: {
        candlestick: {
          colors: { upward: '#26a69a', downward: '#ef5350' },
          wick: { useFillColor: true },
        },
      },
      stroke: {
        show: true,
        curve: 'smooth',
        colors: strokeColors,
        width: strokeWidths,
        dashArray: strokeDashes,
      },
      yaxis: {
        tooltip: { enabled: true },
        labels: { style: { colors: '#888' }, formatter: (v: number) => '$' + v.toLocaleString('en-US', { maximumFractionDigits: 0 }) },
      },
      legend: {
        show: lineNames.length > 0,
        labels: { colors: '#ccc' },
      },
      tooltip: {
        theme: 'dark',
        shared: false,
        custom: undefined,
      },
    };
  }, [indicators]);

  // RSI
  const rsiSeries = React.useMemo(() => [{
    name: 'RSI',
    data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.rsi14 || null })),
  }], [data]);

  const rsiOptions = React.useMemo((): ApexOptions => ({
    ...darkBase,
    chart: { ...darkBase.chart, id: 'rsi-chart', group: 'crypto', type: 'line', height: 130, toolbar: { show: false } },
    stroke: { curve: 'smooth', width: 2, colors: ['#9b59b6'] },
    yaxis: { min: 0, max: 100, tickAmount: 4, labels: { style: { colors: '#888' } } },
    annotations: {
      yaxis: [
        { y: 70, borderColor: '#ef5350', borderWidth: 1, strokeDashArray: 4, label: { text: '70', style: { color: '#ef5350', background: 'transparent' }, position: 'right' } },
        { y: 30, borderColor: '#26a69a', borderWidth: 1, strokeDashArray: 4, label: { text: '30', style: { color: '#26a69a', background: 'transparent' }, position: 'right' } },
      ],
    },
    legend: { show: false },
    tooltip: { theme: 'dark', x: { format: 'yyyy-MM-dd' } },
  }), []);

  // MACD
  const macdSeries = React.useMemo(() => [
    { name: 'MACD', data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.macd || null })) },
    { name: 'Signal', data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.macd_signal || null })) },
  ], [data]);

  const macdOptions = React.useMemo((): ApexOptions => ({
    ...darkBase,
    chart: { ...darkBase.chart, id: 'macd-chart', group: 'crypto', type: 'line', height: 130, toolbar: { show: false } },
    stroke: { curve: 'smooth', width: [2, 2], colors: ['#3498db', '#e67e22'] },
    yaxis: { labels: { style: { colors: '#888' }, formatter: (v: number) => v.toFixed(0) } },
    legend: { show: false },
    tooltip: { theme: 'dark', x: { format: 'yyyy-MM-dd' } },
  }), []);

  // ADX
  const adxSeries = React.useMemo(() => [{
    name: 'ADX',
    data: data.map(d => ({ x: new Date(d.date).getTime(), y: d.adx14 || null })),
  }], [data]);

  const adxOptions = React.useMemo((): ApexOptions => ({
    ...darkBase,
    chart: { ...darkBase.chart, id: 'adx-chart', group: 'crypto', type: 'line', height: 130, toolbar: { show: false } },
    stroke: { curve: 'smooth', width: 2, colors: ['#f06292'] },
    yaxis: { min: 0, max: 100, tickAmount: 4, labels: { style: { colors: '#888' } } },
    annotations: {
      yaxis: [
        { y: 25, borderColor: '#888', borderWidth: 1, strokeDashArray: 4, label: { text: '25', style: { color: '#888', background: 'transparent' }, position: 'right' } },
      ],
    },
    legend: { show: false },
    tooltip: { theme: 'dark', x: { format: 'yyyy-MM-dd' } },
  }), []);

  // ── BB Grid Info ─────────────────────────────────────────────────────────

  const bbInfo = React.useMemo(() => {
    if (!data.length) return { bbUpper: 0, bbLower: 0, bbMiddle: 0 };
    const latest = data[data.length - 1];
    return {
      bbUpper: latest.bb_upper || 0,
      bbLower: latest.bb_lower || 0,
      bbMiddle: latest.bb_middle || 0,
    };
  }, [data]);

  // ── Live price ────────────────────────────────────────────────────────────

  const up = (liveData?.price_change ?? 0) >= 0;
  const nowStr = (() => {
    const t = new Date();
    return `${String(t.getHours()).padStart(2,'0')}:${String(t.getMinutes()).padStart(2,'0')}:${String(t.getSeconds()).padStart(2,'0')}`;
  })();

  const summary = React.useMemo(() => {
    if (!data.length) return { close: '—', high: '—', low: '—', volume: '—' };
    const latest = data[data.length - 1];
    const fmt = (n: number) => n.toLocaleString('en-US', { maximumFractionDigits: 2 });
    return {
      close: '$' + fmt(latest.close),
      high:  '$' + fmt(Math.max(...data.map(d => d.high))),
      low:   '$' + fmt(Math.min(...data.map(d => d.low))),
      volume: fmt(data.reduce((s, d) => s + d.volume, 0)),
    };
  }, [data]);

  // ── Indicator toggle button ───────────────────────────────────────────────

  const indLabel = (key: keyof IndicatorState, label: string, color: string) => (
    <label key={key} style={{ display: 'inline-flex', alignItems: 'center', gap: 5, fontSize: 12, fontWeight: 600, cursor: 'pointer', whiteSpace: 'nowrap', padding: '3px 8px', borderRadius: 4, border: `1px solid ${indicators[key] ? color : 'transparent'}`, color, transition: 'border-color 0.2s' }}>
      <input type="checkbox" checked={indicators[key]} onChange={() => toggleIndicator(key)} style={{ width: 13, height: 13, cursor: 'pointer' }} />
      {label}
    </label>
  );

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div>
      {/* Controls */}
      <section style={{ display: 'flex', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
        {/* Coin tabs */}
        <div style={{ display: 'flex', border: '1px solid #2a2a4a', borderRadius: 8, overflow: 'hidden' }}>
          {(['BTC', 'ETH', 'SOL'] as Coin[]).map((c, i, arr) => (
            <button key={c} onClick={() => handleCoinChange(c)} onMouseDown={(e) => e.preventDefault()}
              style={{ padding: '8px 28px', background: coin === c ? '#e94560' : '#16213e', border: 'none', color: coin === c ? '#fff' : '#888', fontSize: 14, fontWeight: 600, cursor: 'pointer', borderRight: i < arr.length - 1 ? '1px solid #2a2a4a' : 'none' }}>
              {c}
            </button>
          ))}
        </div>

        {/* Period buttons */}
        <div style={{ display: 'flex', border: '1px solid #2a2a4a', borderRadius: 8, overflow: 'hidden' }}>
          {PERIODS.map(p => (
            <button key={p.value} onClick={() => handlePeriodChange(p.value)} onMouseDown={(e) => e.preventDefault()}
              style={{ padding: '8px 14px', background: period === p.value ? '#0f3460' : '#16213e', border: 'none', borderRight: '1px solid #2a2a4a', color: period === p.value ? '#e0e0e0' : '#888', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
              {p.label}
            </button>
          ))}
        </div>

        {/* Custom date range */}
        {period === 'custom' && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#888', fontSize: 13 }}>
              From <input type="date" value={fromDate} onChange={e => setFromDate(e.target.value)}
                style={{ background: '#16213e', border: '1px solid #2a2a4a', borderRadius: 6, color: '#e0e0e0', padding: '7px 10px', fontSize: 13, outline: 'none', colorScheme: 'dark' }} />
            </label>
            <label style={{ display: 'flex', alignItems: 'center', gap: 8, color: '#888', fontSize: 13 }}>
              To <input type="date" value={toDate} onChange={e => setToDate(e.target.value)}
                style={{ background: '#16213e', border: '1px solid #2a2a4a', borderRadius: 6, color: '#e0e0e0', padding: '7px 10px', fontSize: 13, outline: 'none', colorScheme: 'dark' }} />
            </label>
            <button onClick={handleCustomApply}
              style={{ padding: '8px 20px', background: '#0f3460', border: '1px solid #2a2a4a', borderRadius: 6, color: '#e0e0e0', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}>
              조회
            </button>
          </div>
        )}
      </section>

      {/* Indicator bar */}
      <section style={{ display: 'flex', gap: 10, alignItems: 'center', padding: '8px 0', borderBottom: '1px solid #2a2a4a', flexWrap: 'wrap', marginTop: 12 }}>
        {indLabel('ma7',  'MA7',    '#26a69a')}
        {indLabel('ma20', 'MA20',   '#f4d03f')}
        {indLabel('ma50', 'MA50',   '#e74c3c')}
        {indLabel('ema9',  'EMA9',  '#00bcd4')}
        {indLabel('ema21', 'EMA21', '#ff9800')}
        {indLabel('bb',   'BB',     '#7b68ee')}
        {indLabel('rsi',  'RSI',    '#9b59b6')}
        {indLabel('macd', 'MACD',   '#3498db')}
        {indLabel('adx',  'ADX',    '#f06292')}
      </section>

      {/* Live price card */}
      <section style={{ marginTop: 12 }}>
        <div style={{ background: '#16213e', border: '1px solid #2a2a4a', borderRadius: 10, padding: '10px 20px' }}>
          <div style={{ display: 'flex', alignItems: 'baseline', gap: 14, flexWrap: 'wrap' }}>
            <span style={{ fontSize: 30, fontWeight: 800, color: '#fff', lineHeight: 1 }}>
              {liveData ? fmtMoney(liveData.last_price) : '—'}
            </span>
            {liveData && (
              <span style={{ fontSize: 14, fontWeight: 600, color: up ? '#26a69a' : '#ef5350', display: 'flex', alignItems: 'center', gap: 4 }}>
                {up ? '▲' : '▼'} {(up ? '+' : '') + fmtMoney(liveData.price_change)} ({(up ? '+' : '') + liveData.price_change_percent.toFixed(2)}%)
                <span style={{ fontSize: 11, color: '#888', marginLeft: 2 }}>24h</span>
              </span>
            )}
            <span style={{ marginLeft: 'auto', fontSize: 11, fontWeight: 700, color: '#e94560', letterSpacing: 1 }}>● LIVE</span>
          </div>
          <div style={{ display: 'flex', gap: 20, fontSize: 12, color: '#888', flexWrap: 'wrap', marginTop: 6 }}>
            <span>고가 <strong style={{ color: '#26a69a' }}>{liveData ? fmtMoney(liveData.high_price) : '—'}</strong></span>
            <span>저가 <strong style={{ color: '#ef5350' }}>{liveData ? fmtMoney(liveData.low_price) : '—'}</strong></span>
            <span>거래량 <strong style={{ color: '#e0e0e0' }}>{liveData ? liveData.volume.toLocaleString('en-US', { maximumFractionDigits: 2 }) + ' ' + coin : '—'}</strong></span>
            <span>거래대금 <strong style={{ color: '#e0e0e0' }}>{liveData ? formatBigNum(liveData.quote_volume) : '—'}</strong></span>
            <span style={{ marginLeft: 'auto' }}>업데이트 <strong style={{ color: '#666' }}>{liveData ? nowStr : '—'}</strong></span>
          </div>
        </div>
      </section>

      {/* Charts */}
      <section style={{ background: '#1a1a2e', border: '1px solid #2a2a4a', borderRadius: 12, padding: 16, marginTop: 12 }}>
        {data.length === 0 ? (
          <div style={{ height: 460, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#555', fontSize: 14 }}>
            로딩 중...
          </div>
        ) : (
          <>
            <ReactApexChart
              key={`main-${coin}-${period}-${JSON.stringify(indicators)}`}
              options={mainOptions}
              series={mainSeries}
              type="candlestick"
              height={460}
            />

            {indicators.rsi && (
              <div style={{ marginTop: 2 }}>
                <div style={{ padding: '3px 8px', fontSize: 10, color: '#555', background: '#16213e', textTransform: 'uppercase', letterSpacing: '0.08em', borderTop: '1px solid #2a2a4a' }}>RSI (14)</div>
                <ReactApexChart
                  key={`rsi-${coin}-${period}`}
                  options={rsiOptions}
                  series={rsiSeries}
                  type="line"
                  height={130}
                />
              </div>
            )}

            {indicators.macd && (
              <div style={{ marginTop: 2 }}>
                <div style={{ padding: '3px 8px', fontSize: 10, color: '#555', background: '#16213e', textTransform: 'uppercase', letterSpacing: '0.08em', borderTop: '1px solid #2a2a4a' }}>MACD (12, 26, 9)</div>
                <ReactApexChart
                  key={`macd-${coin}-${period}`}
                  options={macdOptions}
                  series={macdSeries}
                  type="line"
                  height={130}
                />
              </div>
            )}

            {indicators.adx && (
              <div style={{ marginTop: 2 }}>
                <div style={{ padding: '3px 8px', fontSize: 10, color: '#555', background: '#16213e', textTransform: 'uppercase', letterSpacing: '0.08em', borderTop: '1px solid #2a2a4a' }}>ADX (14) — 25 이상: 추세 유효</div>
                <ReactApexChart
                  key={`adx-${coin}-${period}`}
                  options={adxOptions}
                  series={adxSeries}
                  type="line"
                  height={130}
                />
              </div>
            )}
          </>
        )}
      </section>

      {/* BB Grid Info Box */}
      {(() => {
        const { bbUpper, bbLower } = bbInfo;
        const currentPrice = liveData?.last_price || 0;
        const targetRatio = bbUpper > bbLower && currentPrice > 0
          ? Math.max(5, Math.min(95, (bbUpper - currentPrice) / (bbUpper - bbLower) * 100))
          : 50;
        return (
          <div style={{ display: 'flex', gap: 16, padding: '8px 12px', background: '#0f0f2f', borderRadius: 6, fontSize: 12, color: '#aaa', marginTop: 8, flexWrap: 'wrap' }}>
            <span>BB 하단 <strong style={{ color: '#26a69a' }}>{bbLower > 0 ? '$' + bbLower.toLocaleString('en-US', { maximumFractionDigits: 0 }) : '—'}</strong></span>
            <span>현재 <strong style={{ color: '#e0e0e0' }}>{currentPrice > 0 ? '$' + currentPrice.toLocaleString('en-US', { maximumFractionDigits: 0 }) : '—'}</strong></span>
            <span>BB 상단 <strong style={{ color: '#ef5350' }}>{bbUpper > 0 ? '$' + bbUpper.toLocaleString('en-US', { maximumFractionDigits: 0 }) : '—'}</strong></span>
            <span>그리드 목표비중 <strong style={{ color: '#ffd54f' }}>{bbUpper > 0 ? targetRatio.toFixed(1) + '%' : '—'}</strong></span>
          </div>
        );
      })()}

      {/* Summary cards */}
      <section style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 16, marginTop: 12 }}>
        {[
          { label: '최신 종가', value: summary.close, color: '#e0e0e0' },
          { label: '고가',     value: summary.high,  color: '#26a69a' },
          { label: '저가',     value: summary.low,   color: '#ef5350' },
          { label: '거래량',   value: summary.volume, color: '#e0e0e0' },
        ].map(card => (
          <div key={card.label} style={{ background: '#16213e', border: '1px solid #2a2a4a', borderRadius: 10, padding: '14px 18px' }}>
            <div style={{ fontSize: 12, color: '#888', textTransform: 'uppercase', letterSpacing: '0.8px', marginBottom: 6 }}>{card.label}</div>
            <div style={{ fontSize: 18, fontWeight: 700, color: card.color }}>{card.value}</div>
          </div>
        ))}
      </section>
    </div>
  );
};

export default ChartPage;
