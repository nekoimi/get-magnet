/**
 * views personal
 */
type NewInfo = {
	title: string;
	date: string;
	link: string;
};
type Recommend = {
	title: string;
	msg: string;
	icon: string;
	bg: string;
	iconColor: string;
};
declare type PersonalState = {
	newsInfoList: NewInfo[];
	recommendList: Recommend[];
	personalForm: {
		name: string;
		email: string;
		autograph: string;
		occupation: string;
		phone: string;
		sex: string;
	};
};

/**
 * views visualizing
 */
declare type Demo2State<T = any> = {
	time: {
		txt: string;
		fun: number;
	};
	dropdownList: T[];
	dropdownActive: string;
	skyList: T[];
	dBtnList: T[];
	chartData4Index: number;
	dBtnActive: number;
	earth3DBtnList: T[];
	chartData4List: T[];
	myCharts: T[];
};

/**
 * views params
 */
declare type ParamsState = {
	value: string;
	tagsViewName: string;
	tagsViewNameIsI18n: boolean;
};

/**
 * views system
 */
// role
declare interface RowRoleType {
	roleName: string;
	roleSign: string;
	describe: string;
	sort: number;
	status: boolean;
	createTime: string;
}

interface SysRoleTableType extends TableType {
	data: RowRoleType[];
}

declare interface SysRoleState {
	tableData: SysRoleTableType;
}

declare type TreeType = {
	id: number;
	label: string;
	children?: TreeType[];
};

// user
declare type RowUserType<T = any> = {
	username: string;
	userNickname: string;
	roleSign: string;
	department: string[];
	phone: string;
	email: string;
	sex: string;
	password: string;
	overdueTime: T;
	status: boolean;
	describe: string;
	createTime: T;
};

interface SysUserTableType extends TableType {
	data: RowUserType[];
}

declare interface SysUserState {
	tableData: SysUserTableType;
}

declare type DeptTreeType = {
	deptName: string;
	createTime: string;
	status: boolean;
	sort: number;
	describe: string;
	id: number | string;
	children?: DeptTreeType[];
};

// dept
declare interface RowDeptType extends DeptTreeType {
	deptLevel: string[];
	person: string;
	phone: string;
	email: string;
}

interface SysDeptTableType extends TableType {
	data: DeptTreeType[];
}

declare interface SysDeptState {
	tableData: SysDeptTableType;
}

// dic
type ListType = {
	id: number;
	label: string;
	value: string;
};

declare interface RowDicType {
	dicName: string;
	fieldName: string;
	describe: string;
	status: boolean;
	createTime: string;
	list: ListType[];
}

interface SysDicTableType extends TableType {
	data: RowDicType[];
}

declare interface SysDicState {
	tableData: SysDicTableType;
}

/**
 * views pages
 */
//  filtering
declare type FilteringChilType = {
	id: number | string;
	label: string;
	active: boolean;
};

declare type FilterListType = {
	img: string;
	title: string;
	evaluate: string;
	collection: string;
	price: string;
	monSales: string;
	id: number | string;
	loading?: boolean;
};

declare type FilteringRowType = {
	title: string;
	isMore: boolean;
	isShowMore: boolean;
	id: number | string;
	children: FilteringChilType[];
};

// tableRules
declare type TableRulesHeaderType = {
	prop: string;
	width: string | number;
	label: string;
	isRequired?: boolean;
	isTooltip?: boolean;
	type: string;
};

declare type TableRulesState = {
	tableData: {
		data: EmptyObjectType[];
		header: TableRulesHeaderType[];
		option: SelectOptionType[];
	};
};

declare type TableRulesOneProps = {
	name: string;
	email: string;
	autograph: string;
	occupation: string;
};

// tree
declare type RowTreeType = {
	id: number;
	label: string;
	label1: string;
	label2: string;
	isShow: boolean;
	children?: RowTreeType[];
};

// workflow index
declare type NodeListState = {
	id: string | number;
	nodeId: string | undefined;
	class: HTMLElement | string;
	left: number | string;
	top: number | string;
	icon: string;
	name: string;
};

declare type LineListState = {
	sourceId: string;
	targetId: string;
	label: string;
};

declare type XyState = {
	x: string | number;
	y: string | number;
};

declare type WorkflowState<T = any> = {
	leftNavList: T[];
	dropdownNode: XyState;
	dropdownLine: XyState;
	isShow: boolean;
	jsPlumb: T;
	jsPlumbNodeIndex: null | number;
	jsplumbDefaults: T;
	jsplumbMakeSource: T;
	jsplumbMakeTarget: T;
	jsplumbConnect: T;
	jsplumbData: {
		nodeList: NodeListState[];
		lineList: LineListState[];
	};
};

// workflow drawer
declare type WorkflowDrawerNodeState<T = any> = {
	node: { [key: string]: T };
	nodeRules: T;
	form: T;
	tabsActive: string;
	loading: {
		extend: boolean;
	};
};

declare type WorkflowDrawerLabelType = {
	type: string;
	label: string;
};

declare type WorkflowDrawerState<T = any> = {
	isOpen: boolean;
	nodeData: {
		type: string;
	};
	jsplumbConn: T;
};

/**
 * views make
 */
// tableDemo
declare type TableDemoPageType = {
	pageNum: number;
	pageSize: number;
};

declare type TableHeaderType = {
	key: string;
	width: string;
	title: string;
	type: string | number;
	colWidth: string;
	width?: string | number;
	height?: string | number;
	isCheck: boolean;
};

declare type TableSearchType = {
	label: string;
	prop: string;
	placeholder: string;
	required: boolean;
	type: string;
	options?: SelectOptionType[];
};

declare type TableDemoState = {
	tableData: {
		data: EmptyObjectType[];
		header: TableHeaderType[];
		config: {
			total: number;
			loading: boolean;
			isBorder: boolean;
			isSelection: boolean;
			isSerialNo: boolean;
			isOperate: boolean;
		};
		search: TableSearchType[];
		param: EmptyObjectType;
		printName: string;
	};
};

/**
 * views magnets
 */
// 磁力链接类型
declare type MagnetType = {
	id: number;
	created_at: string;
	updated_at: string;
	origin: string;
	title: string;
	number: string;
	optimal_link: string;
	links: string[];
	raw_url_host: string;
	raw_url_path: string;
	status: number;
	actress0: string;
	followed_by: string;
	download_error?: string;
	download_retry_count?: number;
	last_submit_at?: string;
	download_completed_at?: string;
	play_file_path?: string;
	play_file_id?: string;
	play_file_size?: number;
	strm_path?: string;
	post_process_done?: boolean;
};

declare type MagnetEventType = {
	id: number;
	created_at: string;
	magnet_id: number;
	event_type: string;
	message?: string;
	extra?: string;
};

declare type MagnetDetailType = {
	magnet: MagnetType;
	status_label: string;
	link_count: number;
	has_play_info: boolean;
	play_url?: string;
	post_process: {
		done: boolean;
		play_file_id?: string;
		play_file_path?: string;
		play_file_size?: number;
		strm_path?: string;
	};
	download: {
		task_id?: string;
		error?: string;
		retry_count: number;
		last_submit_at?: string;
		completed_at?: string;
	};
	events: MagnetEventType[];
	status_options: Array<{
		label: string;
		value: number;
	}>;
};

interface MagnetTableType extends TableType {
	data: MagnetType[];
}

declare interface MagnetState {
	tableData: MagnetTableType;
	actionLoadingIds: number[];
	statusOptions: Array<{
		label: string;
		value: number;
	}>;
	detail: {
		visible: boolean;
		loading: boolean;
		data?: MagnetDetailType;
		cloudTask?: any;
		cloudTaskLoading: boolean;
	};
}
