export const URL = `http://localhost:8080`

export interface RangeCountQueryResult {
	freq: { [key: string]: number }
	total: number
	searchTime: number
}

export async function fetchTimerange(start: Date, end: Date) {
	const req = await fetch(
		URL +
			'/range-count?start=' +
			start.toISOString().slice(0, 10) +
			'&end=' +
			end.toISOString().slice(0, 10)
	)
	const res = (await req.json()) as RangeCountQueryResult
	return res
}
