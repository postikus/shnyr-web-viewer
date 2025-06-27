function openImageModal(imageData, id, info, hasItems, ...items) {
	const modal = document.getElementById('imageModal');
	const modalImage = document.getElementById('modalImage');
	const modalTitle = document.getElementById('modalTitle');
	const modalInfo = document.getElementById('modalInfo');
	const modalStructuredData = document.getElementById('modalStructuredData');
	
	modalImage.src = 'data:image/png;base64,' + imageData;
	modalTitle.textContent = 'ШНЫРЬ НАМУТИЛ СКРИНШОТ #' + id;
	modalInfo.textContent = info;
	
	// Создаем таблицу структурированных данных
	if (hasItems && items.length > 0) {
		// Находим самые дешевые предметы для каждого уровня улучшения
		const enhancementGroups = {};
		items.forEach(item => {
			if (item.enhancement && item.price) {
				const price = parseFloat(item.price.replace(/[^\d.]/g, ''));
				if (!isNaN(price)) {
					if (!enhancementGroups[item.enhancement]) {
						enhancementGroups[item.enhancement] = [];
					}
					enhancementGroups[item.enhancement].push({...item, priceValue: price});
				}
			}
		});
		
		// Находим самые дешевые предметы
		const cheapestItems = new Set();
		Object.values(enhancementGroups).forEach(group => {
			if (group.length > 0) {
				const cheapest = group.reduce((min, item) => 
					item.priceValue < min.priceValue ? item : min
				);
				cheapestItems.add(cheapest);
			}
		});
		
		let tableHTML = '<h4 style="margin: 0 0 10px 0; color: #333; font-size: 1.1em;">📋 Структурированные данные:</h4>';
		tableHTML += '<table class="modal-structured-table">';
		tableHTML += '<tr><th>Title</th><th>Title Short</th><th>Enhancement</th><th>Price</th><th>Count</th><th>Package</th><th>Owner</th><th>Category</th></tr>';
		
		items.forEach(item => {
			const isCheapest = cheapestItems.has(item);
			const rowClass = isCheapest ? 'cheapest-item' : '';
			tableHTML += '<tr class="' + rowClass + '">';
			tableHTML += '<td>' + (item.title || '') + '</td>';
			tableHTML += '<td>' + (item.titleShort || '') + '</td>';
			tableHTML += '<td>' + (item.enhancement || '') + '</td>';
			tableHTML += '<td>' + formatPrice(item.price || '') + '</td>';
			tableHTML += '<td>' + (item.count || '') + '</td>';
			tableHTML += '<td>' + (item.package ? '✔️' : '') + '</td>';
			tableHTML += '<td>' + (item.owner || '') + '</td>';
			tableHTML += '<td>' + formatCategory(item.category || '') + '</td>';
			tableHTML += '</tr>';
		});
		
		tableHTML += '</table>';
		modalStructuredData.innerHTML = tableHTML;
		modalStructuredData.style.display = 'block';
	} else {
		modalStructuredData.innerHTML = '<p style="margin: 0; color: #666; font-style: italic;">Нет структурированных данных</p>';
		modalStructuredData.style.display = 'block';
	}
	
	modal.style.display = 'block';
	document.body.style.overflow = 'hidden'; // Блокируем скролл страницы
}

function closeImageModal() {
	const modal = document.getElementById('imageModal');
	modal.style.display = 'none';
	document.body.style.overflow = 'auto'; // Возвращаем скролл страницы
}

function openDetailModalFromData(element) {
	const rawText = element.getAttribute('data-raw-text');
	const id = element.getAttribute('data-id');
	const imageData = element.getAttribute('data-image');
	const imagePath = element.getAttribute('data-image-path');
	const debugInfo = element.getAttribute('data-debug');
	const hasItems = element.getAttribute('data-items') === 'true';
	const structuredItemsJson = element.getAttribute('data-structured-items');
	
	console.log('openDetailModalFromData called with:', {rawText, id, imageData, imagePath, debugInfo, hasItems, structuredItemsJson});
	
	let items = [];
	if (hasItems && structuredItemsJson) {
		try {
			items = JSON.parse(structuredItemsJson);
			console.log('Parsed items:', items);
		} catch (e) {
			console.error('Error parsing structured items:', e);
			items = [];
		}
	}
	
	openDetailModal(rawText, id, imageData, imagePath, debugInfo, hasItems, ...items);
}

function openDetailModal(text, id, imageData, imagePath, debugInfo, hasItems, ...items) {
	console.log('openDetailModal called with:', {text, id, imageData, imagePath, debugInfo, hasItems, items});
	
	const modalTitle = document.getElementById('detailModalTitle');
	const modalImage = document.getElementById('detailModalImage');
	const modalDebugInfo = document.getElementById('detailModalDebugInfo');
	const modalStructuredData = document.getElementById('detailModalStructuredData');
	const modalRawText = document.getElementById('detailModalRawText');
	const modalImagePath = document.getElementById('detailModalImagePath');
	
	console.log('Found elements:', {
		modalTitle: !!modalTitle,
		modalImage: !!modalImage,
		modalDebugInfo: !!modalDebugInfo,
		modalStructuredData: !!modalStructuredData,
		modalRawText: !!modalRawText,
		modalImagePath: !!modalImagePath
	});
	
	// Устанавливаем заголовок
	modalTitle.textContent = 'ШНЫРЬ НАМУТИЛ СКРИНШОТ #' + id;
	
	// Загружаем изображение
	if (imageData && imageData !== '') {
		modalImage.src = 'data:image/png;base64,' + imageData;
		modalImage.style.display = 'block';
		console.log('Image src set to:', modalImage.src.substring(0, 50) + '...');
	} else {
		modalImage.style.display = 'none';
	}
	
	// Устанавливаем debug info
	modalDebugInfo.textContent = debugInfo || 'Нет debug информации';
	
	// Устанавливаем сырой текст
	modalRawText.textContent = text || 'Нет данных';
	
	// Устанавливаем путь к изображению
	modalImagePath.textContent = imagePath || 'Нет данных';
	
	console.log('hasItems:', hasItems, 'items length:', items.length);
	
	// Обрабатываем структурированные данные
	if (hasItems && items.length > 0) {
		console.log('Processing items:', items);
		const cheapestItems = new Set();
		
		// Группируем предметы по уровню улучшения и package
		const enhancementGroups = {};
		items.forEach(item => {
			const enhancement = item.enhancement || '';
			const isPackage = item.package || false;
			const groupKey = enhancement + '_' + (isPackage ? 'package' : 'nopackage');
			
			if (!enhancementGroups[groupKey]) {
				enhancementGroups[groupKey] = [];
			}
			enhancementGroups[groupKey].push(item);
		});
		
		// Находим самые дешевые предметы в каждой группе
		Object.values(enhancementGroups).forEach(group => {
			if (group.length > 0) {
				const cheapest = group.reduce((min, item) => {
					const priceValue = parseFloat((item.price || '0').replace(/[^\d.]/g, ''));
					const minPriceValue = parseFloat((min.price || '0').replace(/[^\d.]/g, ''));
					return priceValue < minPriceValue ? item : min;
				});
				cheapestItems.add(cheapest);
			}
		});
		
		let tableHTML = '<table class="structured-table">';
		tableHTML += '<thead><tr><th>Название</th><th>Краткое название</th><th>Улучшение</th><th>Цена</th><th>Количество</th><th>Пакет</th><th>Владелец</th><th>Категория</th></tr></thead>';
		tableHTML += '<tbody>';
		
		items.forEach(item => {
			const isCheapest = cheapestItems.has(item);
			const rowClass = isCheapest ? (item.package ? 'cheapest-package' : 'cheapest') : '';
			tableHTML += '<tr class="' + rowClass + '">';
			tableHTML += '<td>' + (item.title || '') + '</td>';
			tableHTML += '<td>' + (item.titleShort || '') + '</td>';
			tableHTML += '<td>' + (item.enhancement || '') + '</td>';
			tableHTML += '<td>' + formatPrice(item.price || '') + '</td>';
			tableHTML += '<td>' + (item.count || '') + '</td>';
			tableHTML += '<td>' + (item.package ? '✔️' : '❌') + '</td>';
			tableHTML += '<td>' + (item.owner || '') + '</td>';
			tableHTML += '<td>' + formatCategory(item.category || '') + '</td>';
			tableHTML += '</tr>';
		});
		
		tableHTML += '</tbody></table>';
		console.log('Generated table HTML:', tableHTML);
		modalStructuredData.innerHTML = tableHTML;
		console.log('Table HTML set to modalStructuredData');
	} else {
		console.log('No items to display');
		modalStructuredData.innerHTML = '<p>Нет структурированных данных</p>';
	}
	
	// Показываем модальное окно
	const detailModal = document.getElementById('detailModal');
	detailModal.style.display = 'block';
	document.body.style.overflow = 'hidden';
	console.log('Modal displayed');
}

function closeDetailModal() {
	const detailModal = document.getElementById('detailModal');
	detailModal.style.display = 'none';
	document.body.style.overflow = 'auto';
}

// Функция для форматирования цены с пробелами
function formatPrice(price) {
	if (!price) return '';
	// Убираем все нецифровые символы
	const cleanPrice = price.replace(/[^\d.]/g, '');
	if (!cleanPrice) return price;
	
	// Добавляем пробелы каждые 3 цифры справа
	let result = '';
	for (let i = 0; i < cleanPrice.length; i++) {
		if (i > 0 && (cleanPrice.length - i) % 3 === 0) {
			result += ' ';
		}
		result += cleanPrice[i];
	}
	return result;
}

// Функция для форматирования категории
function formatCategory(category) {
	if (!category) return '';
	
	switch (category) {
		case 'buy_consumables':
			return '💰 Скупка (расходники)';
		case 'buy_equipment':
			return '💰 Скупка (экипировка)';
		case 'sell_consumables':
			return '💸 Продажа (расходники)';
		case 'sell_equipment':
			return '💸 Продажа (экипировка)';
		case 'unknown':
			return '❓ Неизвестно';
		default:
			return category;
	}
}

// Функция для выделения самых дешевых предметов в главной таблице
function highlightCheapestItems() {
	console.log('highlightCheapestItems called');
	const structuredTables = document.querySelectorAll('.structured-table table');
	console.log('Found structured tables:', structuredTables.length);
	
	structuredTables.forEach(function(table, tableIndex) {
		console.log('Processing table ' + tableIndex);
		const rows = table.querySelectorAll('tr:not(:first-child)'); // Исключаем заголовок
		console.log('Found ' + rows.length + ' data rows');
		const enhancementGroups = {};
		
		// Группируем предметы по уровню улучшения и package
		rows.forEach(function(row, rowIndex) {
			const cells = row.querySelectorAll('td');
			if (cells.length >= 7) { // Обновлено для учета новой колонки категории
				const enhancement = cells[2].textContent.trim();
				const price = cells[3].textContent.trim();
				const package = cells[5].textContent.trim();
				const priceValue = parseFloat(price.replace(/[^\d]/g, ''));
				
				console.log('Row ' + rowIndex + ': enhancement="' + enhancement + '", price="' + price + '", package="' + package + '", priceValue=' + priceValue);
				
				if (enhancement && !isNaN(priceValue)) {
					// Создаем ключ группы: enhancement + package
					const groupKey = enhancement + '_' + (package.includes('✔️') ? 'package' : 'nopackage');
					if (!enhancementGroups[groupKey]) {
						enhancementGroups[groupKey] = [];
					}
					enhancementGroups[groupKey].push({
						row: row, 
						priceValue: priceValue, 
						isPackage: package.includes('✔️')
					});
				}
			}
		});
		
		console.log('Enhancement groups:', enhancementGroups);
		
		// Находим и выделяем самые дешевые предметы в каждой группе
		Object.keys(enhancementGroups).forEach(function(groupKey) {
			const group = enhancementGroups[groupKey];
			console.log('Processing group ' + groupKey + ' with ' + group.length + ' items');
			if (group.length > 0) {
				const cheapest = group.reduce(function(min, item) {
					return item.priceValue < min.priceValue ? item : min;
				});
				console.log('Cheapest in group ' + groupKey + ': price=' + cheapest.priceValue + ', isPackage=' + cheapest.isPackage);
				
				// Применяем соответствующий класс в зависимости от package
				if (cheapest.isPackage) {
					cheapest.row.classList.add('cheapest-package');
				} else {
					cheapest.row.classList.add('cheapest');
				}
			}
		});
	});
}

// Инициализация при загрузке страницы
document.addEventListener('DOMContentLoaded', function() {
	// Вызываем функцию выделения дешевых предметов
	highlightCheapestItems();
	
	// Закрытие модального окна при клике вне изображения
	document.getElementById('imageModal').addEventListener('click', function(e) {
		if (e.target === this) {
			closeImageModal();
		}
	});
	
	// Закрытие модального окна по клавише Escape
	document.addEventListener('keydown', function(e) {
		if (e.key === 'Escape') {
			closeImageModal();
			closeDetailModal();
		}
	});
	
	// Закрытие модального окна при клике вне информации
	document.getElementById('detailModal').addEventListener('click', function(e) {
		if (e.target === this) {
			closeDetailModal();
		}
	});
});

function formatDateTime(dateTimeStr) {
	const date = new Date(dateTimeStr);
	// Добавляем 8 часов (UTC+8)
	date.setHours(date.getHours() + 8);
	
	const day = String(date.getDate()).padStart(2, '0');
	const month = String(date.getMonth() + 1).padStart(2, '0');
	const year = date.getFullYear();
	const hours = String(date.getHours()).padStart(2, '0');
	const minutes = String(date.getMinutes()).padStart(2, '0');
	const seconds = String(date.getSeconds()).padStart(2, '0');
	
	return `${day}.${month}.${year} ${hours}:${minutes}:${seconds}`;
}

function sendAction(action) {
	console.log('Отправляем действие:', action);
	
	fetch('/' + action, {
		method: 'POST',
		headers: {
			'Content-Type': 'application/json',
		}
	})
	.then(response => {
		if (response.ok) {
			console.log('Действие', action, 'успешно отправлено');
			// Обновляем только статус через 1 секунду, без перезагрузки страницы
			setTimeout(() => {
				// Вызываем updateStatus если она доступна
				if (typeof updateStatus === 'function') {
					updateStatus();
				}
			}, 1000);
		} else {
			console.error('Ошибка при отправке действия:', response.status);
			alert('Ошибка при отправке действия: ' + response.status);
		}
	})
	.catch(error => {
		console.error('Ошибка сети:', error);
		alert('Ошибка сети при отправке действия');
	});
} 