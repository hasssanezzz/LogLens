import { fetchTimerange } from '~/helpers/fetcher'
import type { Route } from './+types/home'
import {
	ResponsiveContainer,
	BarChart,
	CartesianGrid,
	XAxis,
	YAxis,
	Tooltip,
	Legend,
	Bar,
	Rectangle,
} from 'recharts'
import { useEffect, useState } from 'react'

export async function clientLoader() {
	const today = new Date()
	const res = await fetchTimerange(
		new Date(today.getFullYear(), today.getMonth() - 1),
		today
	)
	return res
}

const CustomTooltip = ({ payload, label }: any) => {
	if (payload && payload.length) {
		return (
			<div className="custom-tooltip bg-gray-700 p-3 shadow-lg rounded-md">
				<p>
					<span className="text-gray-400">Date:</span> {payload[0].payload.date}
				</p>
				<p>
					<span className="text-gray-400">Frequency:</span>{' '}
					{payload[0].payload.count}
				</p>
			</div>
		)
	}
	return null
}

export default function Home({ loaderData }: Route.ComponentProps) {
	const res = loaderData
	const [currRange, setCurrRange] = useState<'7' | '14' | '30' | 'other'>('30')
	const lastMonth = Object.entries(res.freq).map(([date, count]) => ({
		date,
		count,
	}))
	const [data, setData] = useState(lastMonth)

	useEffect(() => {
		if (currRange !== 'other') {
			setData((d) =>
				lastMonth.slice(lastMonth.length - +currRange, lastMonth.length)
			)
		}
	}, [currRange])

	return (
		<div className="cont pt-10">
			<main className="bg-gray-900 rounded-md shadow-md px-5 pb-10 space-y-5">
				<nav className="w-full px-5 h-24 flex items-center justify-between">
					<div className="flex flex-col">
						<span className="font-bold text-3xl text-center">{res.total}</span>
						<small>in the last month</small>
					</div>
					<div className="space-x-3">
						<button
							onClick={() => setCurrRange('7')}
							className={`px-5 py-2 rounded-md shadow-md ${
								currRange === '7'
									? 'bg-blue-500'
									: 'bg-gray-500 hover:bg-gray-400'
							}`}
						>
							7 days
						</button>
						<button
							onClick={() => setCurrRange('14')}
							className={`px-5 py-2 rounded-md shadow-md ${
								currRange === '14'
									? 'bg-blue-500'
									: 'bg-gray-500 hover:bg-gray-400'
							}`}
						>
							14 days
						</button>
						<button
							onClick={() => setCurrRange('30')}
							className={`px-5 py-2 rounded-md shadow-md ${
								currRange === '30'
									? 'bg-blue-500'
									: 'bg-gray-500 hover:bg-gray-400'
							}`}
						>
							30 days
						</button>
					</div>
				</nav>

				<div className="h-[250px]">
					<ResponsiveContainer width="100%" height="100%">
						<BarChart data={data}>
							<XAxis dataKey="date" />
							<Tooltip content={<CustomTooltip />} />
							<Bar
								dataKey="count"
								fill="#2c80ff"
								activeBar={<Rectangle fill="#2c80ff" opacity={0.8} />}
							/>
						</BarChart>
					</ResponsiveContainer>
				</div>
			</main>
		</div>
	)
}
