import { Link } from 'react-router'
import { FaGithub, FaCog } from 'react-icons/fa'
import { HiOutlineCog6Tooth } from 'react-icons/hi2'

export function Navbar() {
	return (
		<nav className="w-full h-20 bg-gray-700 shadow">
			<div className="cont h-full flex items-center justify-between">
				<Link to="/" className="text-2xl font-bold">
					LogLens
				</Link>

				<div className="flex items-center gap-5">
					<Link
						to="/settings"
					>
						<HiOutlineCog6Tooth size={30} />
					</Link>
					<a href="https://github.com/hasssanezzz/loglens" target='_blank'>
						<FaGithub size={30} />
					</a>
				</div>
			</div>
		</nav>
	)
}
